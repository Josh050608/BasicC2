//go:build !windows
// +build !windows

package persistence

import "fmt"

// installImpl 在非 Windows 平台上返回错误
func installImpl() error {
	return fmt.Errorf("persistence is only supported on Windows")
}
