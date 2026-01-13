package lateral

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// MoveMethod 横向移动方法类型
type MoveMethod string

const (
	// MethodWMI 使用 WMI 进行横向移动
	MethodWMI MoveMethod = "wmi"
	// MethodPsExec 使用 PsExec 进行横向移动
	MethodPsExec MoveMethod = "psexec"
	// MethodSMB 使用 SMB 进行横向移动
	MethodSMB MoveMethod = "smb"
	// MethodWinRM 使用 WinRM 进行横向移动
	MethodWinRM MoveMethod = "winrm"
	// MethodSchtasks 使用计划任务进行横向移动
	MethodSchtasks MoveMethod = "schtasks"
)

// Credentials 凭证信息
type Credentials struct {
	Username string
	Password string
	Domain   string
	Hash     string // NTLM Hash for PTH
}

// Target 目标主机信息
type Target struct {
	IP       string
	Hostname string
	Port     int
}

// MoveRequest 横向移动请求
type MoveRequest struct {
	Method      MoveMethod
	Target      Target
	Credentials Credentials
	Command     string // 要在目标机器上执行的命令
	PayloadPath string // Payload 文件路径
}

// MoveResult 横向移动结果
type MoveResult struct {
	Success bool
	Method  MoveMethod
	Target  string
	Message string
	Output  string
	Error   error
}

// LateralMover 横向移动管理器
type LateralMover struct {
	timeout time.Duration
}

// NewLateralMover 创建横向移动管理器
func NewLateralMover() *LateralMover {
	return &LateralMover{
		timeout: 60 * time.Second,
	}
}

// Move 执行横向移动
func (lm *LateralMover) Move(req MoveRequest) MoveResult {
	switch req.Method {
	case MethodWMI:
		return lm.moveViaWMI(req)
	case MethodPsExec:
		return lm.moveViaPsExec(req)
	case MethodSMB:
		return lm.moveViaSMB(req)
	case MethodWinRM:
		return lm.moveViaWinRM(req)
	case MethodSchtasks:
		return lm.moveViaSchtasks(req)
	default:
		return MoveResult{
			Success: false,
			Method:  req.Method,
			Target:  req.Target.IP,
			Error:   fmt.Errorf("unsupported lateral movement method: %s", req.Method),
		}
	}
}

// SetTimeout 设置超时时间
func (lm *LateralMover) SetTimeout(timeout time.Duration) {
	lm.timeout = timeout
}

// execCommand 执行命令并返回输出
func (lm *LateralMover) execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// buildCredString 构建凭证字符串
func buildCredString(creds Credentials) string {
	if creds.Domain != "" {
		return fmt.Sprintf("%s\\%s", creds.Domain, creds.Username)
	}
	return creds.Username
}

// validateRequest 验证请求参数
func validateRequest(req MoveRequest) error {
	if req.Target.IP == "" && req.Target.Hostname == "" {
		return fmt.Errorf("target IP or hostname is required")
	}
	if req.Credentials.Username == "" {
		return fmt.Errorf("username is required")
	}
	if req.Credentials.Password == "" && req.Credentials.Hash == "" {
		return fmt.Errorf("password or hash is required")
	}
	return nil
}

// getTargetAddress 获取目标地址
func getTargetAddress(target Target) string {
	if target.IP != "" {
		return target.IP
	}
	return target.Hostname
}

// sanitizeOutput 清理输出内容
func sanitizeOutput(output string) string {
	// 移除敏感信息
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.TrimSpace(output)
	return output
}
