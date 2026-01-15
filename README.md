# BasicC2 - 模块化 C2 框架

## 项目结构

```
BasicC2/
├── cmd/                          # 主程序入口
│   ├── server/                   # C2 服务器
│   │   └── main.go
│   ├── agent/                    # 被控端
│   │   └── main.go
│   └── loader/                   # 加载器
│       └── main.go
├── internal/                     # 内部共享包
│   ├── crypto/                   # 加密模块
│   │   └── aes.go
│   ├── config/                   # 配置管理
│   │   └── config.go
│   ├── models/                   # 数据结构
│   │   └── types.go
│   └── dga/                      # DGA 域名生成
│       └── dga.go
├── agent/                        # Agent 特有模块
│   ├── commands/                 # 命令执行
│   │   ├── executor.go
│   │   ├── screenshot.go
│   │   └── lateral.go           # 横向移动命令处理
│   ├── lateral/                  # 横向移动模块 ⭐新增
│   │   ├── lateral.go           # 核心管理器
│   │   ├── wmi.go               # WMI 横向移动
│   │   ├── psexec.go            # PsExec 横向移动
│   │   ├── smb.go               # SMB 横向移动
│   │   ├── schtasks.go          # 计划任务横向移动
│   │   ├── recon.go             # 侦察功能
│   │   ├── lateral_test.go      # 单元测试
│   │   └── README.md            # 模块文档
│   ├── persistence/              # 持久化
│   │   └── install.go
│   └── evasion/                  # 反沙箱/反检测
│       └── sandbox.go
├── server/                       # Server 特有模块
│   ├── handlers/                 # HTTP 处理器
│   │   └── handlers.go
│   └── storage/                  # 数据存储
│       └── memory.go
├── loader/                       # Loader 特有模块
│   ├── fetch/                    # Payload 下载
│   │   └── downloader.go
│   └── inject/                   # 内存注入
│       └── injector.go
├── web/                          # Web 前端
│   └── index.html
├── .vscode/                      # VS Code 配置
│   └── settings.json            # 编辑器设置
├── go.mod                        # Go 模块定义
├── go.sum                        # 依赖版本锁定
├── Makefile                      # 构建脚本
└── README.md                     # 项目文档
```

## 核心功能

### 1. C2 服务器 (Server)
- ✅ 加密通信 (AES-GCM)
- ✅ 心跳机制与主机管理
- ✅ 命令下发与回显
- ✅ Web 控制台界面
- ✅ RESTful API

### 2. 被控端 (Agent)
- ✅ 反沙箱检测
- ✅ 自动持久化 (注册表)
- ✅ DGA 域名生成
- ✅ 命令执行
- ✅ 屏幕截图
- ✅ 中文编码处理
- ✅ 横向移动 (Lateral Movement)
- ✅ 网络侦察与信息收集

### 3. 加载器 (Loader)
- ✅ DGA Payload 下载
- ✅ 内存注入执行
- ✅ 无文件落地 (Shellcode)

## 快速开始

### 编译与部署

本项目使用 Makefile 进行编译，支持在 Linux 和 macOS 上构建所有组件。

**详细编译指南请参考：[BUILD.md](BUILD.md)**

简要步骤：
1.  准备 Go 1.20+ 环境。
2.  运行 `make all` 编译所有组件。
3.  运行 `make help` 查看所有可用命令。
4.  编译产物生成在 `build/` 目录下。

### 运行

1. **启动 Server**
```bash
# 在 Linux (ARM) 服务器上运行
./build/server
# 访问 http://ip:8080 查看 Web 控制台
```

2. **生成与测试**
   - `build/agent.exe` 是原始后门程序，用于生成 Shellcode 或测试功能。
   - `build/loader.exe` 是最终投放的加载器（通常不需要参数直接运行）。

3. **运行 Loader**
```bash
# 在目标 Windows 机器上执行
loader.exe
```

## 配置说明

所有配置集中在 [internal/config/config.go](internal/config/config.go)：

```go
// 加密密钥（Server 和 Agent 必须一致）
var AESKey = []byte("HereIsMySecretKey123456789012345")

// C2 服务器地址
const BackupC2URL = "https://api.cailiu666.xyz"

// DGA 种子、截图和横向移动
- **lateral**: 横向移动模块，支持 WMI、PsExec、SMB、WinRM、计划任务等多种方法
- **persistence**: 持久化安装（注册表 + 文件复制）
- **evasion**: 反沙箱检测（CPU 核心数等）

### 横向移动模块特性

支持多种横向移动技术：
- **WMI**: 使用 Windows Management Instrumentation
- **PsExec**: 使用 Sysinternals PsExec 工具
- **SMB**: 通过 SMB 共享复制和执行文件
- **WinRM**: 使用 PowerShell Remoting
- **Schtasks**: 通过计划任务执行

侦察功能：
- 网络扫描 (主机发现)
- SMB/WMI/WinRM 访问权限检查
- 系统信息收集
- 进程列表获取
- 用户列表获取

详细文档请参考：[agent/lateral/README.md](agent/lateral/README.md)
## 模块说明

### internal/ - 公共模块
- **crypto**: 提供 AES-GCM 加密/解密函数
- **config**: 全局配置常量
- **models**: 共享的数据结构

### agent/ - Agent 模块
- **commands**: 命令执行器，支持 CMD 命令和截图
- **persistence**: 持久化安装（注册表 + 文件复制）
- **evasion**: 反沙箱检测（CPU 核心数等）
- **dga**: DGA 域名生成与 C2 协商

### server/ - Server 模块
- **handlers**: HTTP 请求处理器
- **storage**: 内存数据库（主机列表、命令队列）

### loader/ - Loader 模块
- **fetch**: Payload 下载器（支持 DGA）
- **inject**: 内存注入与执行

## 扩展新功能

### 添加新的 Agent 命令

1. 在 [agent/commands/](agent/commands/) 创建新文件，例如 `upload.go`
2. 实现命令逻辑
3. 在 [executor.go](agent/commands/executor.go) 的 `Execute()` 函数中添加分支

### 添加新的反检测手段

1. 在 [agent/evasion/](agent/evasion/) 创建新文件，例如 `debugger.go`
2. 实现检测逻辑
3. 在 [cmd/agent/main.go](cmd/agent/main.go) 的 `main()` 函数中调用

### 添加数据库持久化

1. 在 [server/storage/](server/storage/) 创建 `database.go`
2. 实现 SQL/NoSQL 存储接口
3. 修改 [cmd/server/main.go](cmd/server/main.go) 使用新的存储后端

## 技术栈

- **语言**: Go 1.24
- **加密**: AES-GCM (crypto/aes)
- **截图**: github.com/kbinani/screenshot
- **编码**: golang.org/x/text (GBK/UTF-8)
- **系统调用**: golang.org/x/sys/windows

## 安全警告

⚠️ 本项目仅供教育和研究目的使用。请勿用于非法用途。

## 许可证

本项目为课程设计作业，版权归作者所有。
