package persistence

// Install 实现持久化（仅在 Windows 上有效）
// 在非 Windows 平台上，此函数返回错误
func Install() error {
	return installImpl()
}
