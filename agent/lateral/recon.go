package lateral

import (
	"fmt"
	"strings"
)

// ScanNetwork 扫描网络中的存活主机（分批并行，兼容 PowerShell 5.1）
func ScanNetwork(subnet string) ([]string, error) {
	// 分批扫描，每批 20 个 IP，避免创建过多 Jobs
	psScript := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$subnet = "%s"
		$allResults = @()
		
		# 分批扫描，每批 20 个
		for ($batch = 0; $batch -lt 13; $batch++) {
			$start = $batch * 20 + 1
			$end = [Math]::Min(($batch + 1) * 20, 254)
			
			$jobs = @()
			for ($i = $start; $i -le $end; $i++) {
				$ip = "$subnet.$i"
				$jobs += Start-Job -ScriptBlock {
					param($target)
					if (Test-Connection -ComputerName $target -Count 1 -Quiet) {
						$target
					}
				} -ArgumentList $ip
			}
			
			# 等待当前批次完成
			Wait-Job $jobs -Timeout 5 | Out-Null
			$allResults += $jobs | Receive-Job
			$jobs | Remove-Job -Force
		}
		
		$allResults -join ","
	`, subnet)

	args := []string{
		"-ExecutionPolicy", "Bypass",
		"-NoProfile",
		"-Command", psScript,
	}

	lm := NewLateralMover()
	output, err := lm.execCommand("powershell.exe", args...)
	if err != nil {
		return nil, err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return []string{}, nil
	}

	ips := strings.Split(output, ",")

	// 过滤空字符串
	var result []string
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			result = append(result, ip)
		}
	}

	return result, nil
}

// ScanNetworkSimple 简单快速扫描（最稳定，推荐）
func ScanNetworkSimple(subnet string) ([]string, error) {
	// 使用最简单的方法，一次扫描 10 个常用 IP
	commonIPs := []int{1, 10, 50, 100, 101, 150, 200, 220, 221, 254}

	psScript := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$subnet = "%s"
		$targets = @(%s)
		$results = @()
		
		foreach ($i in $targets) {
			$ip = "$subnet.$i"
			if (Test-Connection -ComputerName $ip -Count 1 -Quiet) {
				$results += $ip
			}
		}
		
		$results -join ","
	`, subnet, strings.Trim(strings.Join(strings.Fields(fmt.Sprint(commonIPs)), ","), "[]"))

	args := []string{
		"-ExecutionPolicy", "Bypass",
		"-NoProfile",
		"-Command", psScript,
	}

	lm := NewLateralMover()
	output, err := lm.execCommand("powershell.exe", args...)
	if err != nil {
		return nil, err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return []string{}, nil
	}

	ips := strings.Split(output, ",")

	var result []string
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			result = append(result, ip)
		}
	}

	return result, nil
}

// CheckSMBAccess 检查 SMB 访问权限
func CheckSMBAccess(target Target, creds Credentials) bool {
	lm := NewLateralMover()
	targetAddr := getTargetAddress(target)
	credStr := buildCredString(creds)

	remotePath := fmt.Sprintf("\\\\%s\\IPC$", targetAddr)
	args := []string{
		"use",
		remotePath,
		"/user:" + credStr,
		creds.Password,
	}

	_, err := lm.execCommand("net", args...)
	if err == nil {
		lm.execCommand("net", "use", remotePath, "/delete", "/y")
		return true
	}

	return false
}

// CheckWMIAccess 检查 WMI 访问权限
func CheckWMIAccess(target Target, creds Credentials) bool {
	lm := NewLateralMover()
	targetAddr := getTargetAddress(target)
	credStr := buildCredString(creds)

	args := []string{
		"/node:" + targetAddr,
		"/user:" + credStr,
		"/password:" + creds.Password,
		"os",
		"get",
		"Caption",
	}

	output, err := lm.execCommand("wmic", args...)
	return err == nil && !contains(output, "Invalid")
}

// CheckWinRMAccess 检查 WinRM 访问权限
func CheckWinRMAccess(target Target, creds Credentials) bool {
	lm := NewLateralMover()
	targetAddr := getTargetAddress(target)
	credStr := buildCredString(creds)

	psScript := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$pass = ConvertTo-SecureString "%s" -AsPlainText -Force;
		$cred = New-Object System.Management.Automation.PSCredential("%s", $pass);
		$option = New-PSSessionOption -SkipCACheck -SkipCNCheck -SkipRevocationCheck;
		try {
			Test-WSMan -ComputerName %s -Credential $cred -ErrorAction Stop
			Write-Output "Success"
		} catch {
			Write-Output "Failed"
		}
	`, creds.Password, credStr, targetAddr)

	args := []string{
		"-ExecutionPolicy", "Bypass",
		"-NoProfile",
		"-Command", psScript,
	}

	output, err := lm.execCommand("powershell.exe", args...)
	return err == nil && contains(output, "Success")
}

// GetSystemInfo 获取目标系统信息
func GetSystemInfo(target Target, creds Credentials) (string, error) {
	lm := NewLateralMover()
	targetAddr := getTargetAddress(target)
	credStr := buildCredString(creds)

	args := []string{
		"/node:" + targetAddr,
		"/user:" + credStr,
		"/password:" + creds.Password,
		"computersystem",
		"get",
		"name,domain,manufacturer,model",
		"/format:list",
	}

	output, err := lm.execCommand("wmic", args...)
	return sanitizeOutput(output), err
}

// ListProcesses 列出目标主机上的进程
func ListProcesses(target Target, creds Credentials) (string, error) {
	lm := NewLateralMover()
	targetAddr := getTargetAddress(target)
	credStr := buildCredString(creds)

	args := []string{
		"/node:" + targetAddr,
		"/user:" + credStr,
		"/password:" + creds.Password,
		"process",
		"list",
		"brief",
	}

	output, err := lm.execCommand("wmic", args...)
	return sanitizeOutput(output), err
}

// ListUsers 列出目标主机上的用户
func ListUsers(target Target, creds Credentials) (string, error) {
	lm := NewLateralMover()
	targetAddr := getTargetAddress(target)
	credStr := buildCredString(creds)

	psScript := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$pass = ConvertTo-SecureString "%s" -AsPlainText -Force;
		$cred = New-Object System.Management.Automation.PSCredential("%s", $pass);
		$option = New-PSSessionOption -SkipCACheck -SkipCNCheck -SkipRevocationCheck;
		Invoke-Command -ComputerName %s -Credential $cred -SessionOption $option -ScriptBlock { Get-LocalUser | Select-Object Name, Enabled, Description }
	`, creds.Password, credStr, targetAddr)

	args := []string{
		"-ExecutionPolicy", "Bypass",
		"-NoProfile",
		"-Command", psScript,
	}

	output, err := lm.execCommand("powershell.exe", args...)
	return sanitizeOutput(output), err
}
