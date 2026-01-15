package evasion

import "runtime"

// CheckSandbox 检查是否运行在沙箱/虚拟机环境中
// 返回 true 表示环境可疑，应退出
func CheckSandbox() bool {
	// 检查 CPU 核心数
	// 大多数沙箱分配的 CPU 核心数很少 (< 2)
	if runtime.NumCPU() < 2 {
		return true
	}

	// TODO: 可以添加更多检测项
	// - MAC 地址前缀检测
	// - 硬盘大小检测
	// - 运行时间检测
	// - 特定进程检测（调试器、监控工具）
	
	return false
}
