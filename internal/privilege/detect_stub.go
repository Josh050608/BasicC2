//go:build !windows
// +build !windows

package privilege

// IsAdmin 非 Windows 平台始终返回 false
func IsAdmin() bool {
	return false
}

// GetIntegrityLevel 非 Windows 平台返回 Unknown
func GetIntegrityLevel() string {
	return "Unknown"
}
