//go:build !windows
// +build !windows

package privilege

import "fmt"

// ElevateViaFodhelper 非 Windows 平台不支持
func ElevateViaFodhelper(payload string) error {
	return fmt.Errorf("fodhelper UAC bypass is only supported on Windows")
}

// CheckFodhelperAvailable 非 Windows 平台返回 false
func CheckFodhelperAvailable() bool {
	return false
}
