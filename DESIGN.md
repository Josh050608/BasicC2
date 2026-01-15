# BasicC2 系统详细设计报告

## 1. 概述 (Overview)

BasicC2 是一个模块化、轻量级的命令与控制（Command & Control, C2）框架，旨在提供隐蔽的远程管理与后渗透测试能力。本项目采用 Go 语言开发，具备跨平台编译能力，核心架构分为服务端（Server）、被控端（Agent）和加载器（Loader）三个部分。

设计目标重点在于：
- **隐蔽性**：通过加密通信、内存注入和反沙箱技术规避检测。
- **韧性**：使用 DGA（域名生成算法）和备用信道确保控制链路的稳定性。
- **扩展性**：模块化的横向移动和功能插件设计，便于二次开发。

## 2. 系统架构 (System Architecture)

系统采用经典的 C/S（Client-Server）架构，由以下核心组件构成：

```mermaid
graph TD
    Operator[攻击者/Operator] -->|HTTP/Web Console| Server[C2 Server (Linux)]
    
    subgraph Victim Environment [受害者网络环境]
        Loader[Loader (Windows)] -->|HTTPS/DGA| Server
        Agent[Agent (Windows)] -->|AES Encrypted HTTP| Server
        
        Agent -->|Lateral Movement| Target1[横向目标 1]
        Agent -->|Lateral Movement| Target2[横向目标 2]
    end
```

### 2.1 组件说明

| 组件 | 运行环境 | 职责 |
|------|----------|------|
| **Server** | Linux (ARM64) | 维护主机列表，下发指令，接收回显，提供 Web 控制台和 API 接口。 |
| **Agent** | Windows (AMD64) | 驻留被控主机，执行命令，收集信息，进行横向移动与持久化。 |
| **Loader** | Windows (AMD64) | 负责从远程下载 Agent 载荷（Payload），并将其注入到系统进程中无文件运行。 |

## 3. 模块详细设计 (Module Design)

### 3.1 C2 Server (控制端)

服务端是整个系统的核心大脑，主要由以下子模块组成：

*   **HTTP 处理器 (`handlers`)**:
    *   处理 Agent 的心跳请求（Heartbeat）。
    *   解析并验证加密的通信数据。
    *   提供 REST API 供 Operator 查看状态和下发指令。
*   **内存存储 (`storage`)**:
    *   使用 `Memory` 结构体在内存中维护所有 Agent 的状态（ID、IP、最后活跃时间、待执行命令队列）。
    *   *注：当前版本为易失性存储，重启服务后状态重置。*
*   **Web 界面**:
    *   提供可视化的主机列表和命令终端（`web/index.html`）。

### 3.2 Agent (被控端)

Agent 作为一个常驻后台的程序，负责执行具体任务：

*   **主循环 (`main.go`)**: 定期（Jitter 随机化）向 Server 发送心跳包，获取指令。
*   **命令执行器 (`commands/executor.go`)**:
    *   解析指令（普通 Shell 命令或内置特殊指令）。
    *   封装 `cmd.exe` 或 PowerShell 执行系统命令。
*   **横向移动模块 (`agent/lateral`)**:
    *   集成多种协议：WMI, PsExec, SMB, WinRM, Schtasks。
    *   支持哈希传递（Pass-The-Hash）和明文凭证认证。
*   **持久化 (`persistence`)**:
    *   通过修改 Windows 注册表启动项（Run Keys）实现开机自启。
    *   伪装名称：`MicrosoftSystemUpdate` (`sys_update.exe`)。
*   **反检测 (`evasion`)**:
    *   **反沙箱**: 检测 CPU 核心数等环境特征。

### 3.3 Loader (加载器)

Loader 的设计目标是确保存活并将 Agent 植入内存：

*   **DGA 域名生成 (`internal/dga`)**:
    *   基于日期和种子 (`MySecretSeed2024`) 生成动态 C2 域名列表，防止域名黑名单封锁。
    *   具备保底机制：DGA 失败时回退到硬编码 IP/域名。
*   **Payload 下载 (`loader/fetch`)**:
    *   从计算出的 URL 下载加密的 Payload (Shellcode/Binary)。
*   **进程注入 (`loader/inject`)**:
    *   **Direct Syscalls**: 使用直接系统调用（`NtAllocateVirtualMemory`, `NtWriteVirtualMemory` 等）绕过用户态 EDR 挂钩（User-land Hooks）。
    *   **目标选择**: 寻找合适的宿主进程（如 `spoolsv.exe`）。
    *   **注入流程**: 提权 (SeDebugPrivilege) -> 打开进程 -> 申请内存 -> 写入 Shellcode -> 创建远程线程。

## 4. 通信协议 (Communication Protocol)

Server 与 Agent 之间采用自定义的加密 HTTP 协议。

### 4.1 加密层
*   **算法**: AES-GCM (Galois/Counter Mode)。
*   **密钥**: 预共享密钥 (Hardcoded Key in `config.go`)。
*   **流程**: 所有 JSON 数据字段（如指令内容 `Data`，回显结果 `Token`）均先经过 AES 加密再进行 Base64 编码传输。

### 4.2 数据包结构

**Agent -> Server (请求)**
```json
{
  "hostname": "Win-User-PC",
  "token": "BASE64(AES(Exec_Result))",  // 加密的回显结果
  "status": "active"
}
```

**Server -> Agent (响应)**
```json
{
  "code": 200,
  "data": "BASE64(AES(cmd /c whoami))" // 加密的待执行指令
}
```

## 5. 核心流程 (Core Workflows)

### 5.1 上线流程
1.  **Loader 启动** -> 检测环境（反沙箱）。
2.  **计算 DGA** -> 尝试连接生成的域名下载 Payload。
3.  **下载成功** -> 解密 Payload。
4.  **进程注入** -> 将 Payload 注入系统进程（如 `spoolsv.exe`）。
5.  **Agent 运行** -> 初始化配置，向 Server 发送第一次心跳，注册主机。

### 5.2 命令执行流程
1.  **Operator** 通过 Web 界面提交命令（如 `dir`）。
2.  **Server** 将命令存入该 Agent 的待执行队列。
3.  **Agent** 下一次心跳轮询时，Server 取出命令，加密后放入响应包返回。
4.  **Agent** 解密命令 -> 调用 `executor` 执行 -> 获取输出。
5.  **Agent** 将输出加密，放入下一次心跳包的 `token` 字段回传。
6.  **Server** 接收并解密回显，展示在前端。

### 5.3 横向移动流程
1.  Agent 接收到 `lateral_move` 指令（包含目标 IP、凭证、方法）。
2.  调用 `agent/lateral` 子模块。
3.  根据指定方法（如 WMI）构建远程执行上下文。
4.  在目标机器上执行 Payload 或命令。
5.  返回执行结果（成功/失败）。

## 6. 数据模型 (Data Model)

主要数据结构定义在 `internal/models/types.go`：

*   **AgentInfo**: 描述受控主机元数据（ID, IP, Hostname, LastSeen）。
*   **CommandRequest**: 描述 Operator 的指令。
*   **LateralMoveRequest**: 描述横向移动的详细参数（Method, Target, Credentials）。

---
*本文档基于当前代码库（2024-2026）的实现状态编写。*
