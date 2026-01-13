//go:build windows
// +build windows

package privilege

import (
	"golang.org/x/sys/windows"
)

// IsAdmin 检测当前是否具有管理员权限
func IsAdmin() bool {
	var sid *windows.SID

	// 获取 Administrators 组的 SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	// 检查当前进程是否属于该组
	member, err := windows.Token(0).IsMember(sid)
	if err != nil {
		return false
	}

	return member
}

// GetIntegrityLevel 获取完整性级别
func GetIntegrityLevel() string {
	// 简化实现：直接使用 IsAdmin() 判断
	if IsAdmin() {
		return "High"
	}
	return "Medium"
}
