# Makefile for BasicC2

# 变量定义
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test

# 输出目录
BUILD_DIR=build

# 服务器配置
SERVER_NAME=server
SERVER_MAIN=./cmd/server/main.go

# Agent 配置 (Windows)
AGENT_NAME=agent.exe
AGENT_MAIN=./cmd/agent/main.go

# Loader 配置 (Windows)
LOADER_NAME=loader.exe
LOADER_MAIN=./cmd/loader/main.go

# 交叉编译环境变量
# Server 目标: Ubuntu 22.04 ARM64
LINUX_ENV=GOOS=linux GOARCH=arm64 CGO_ENABLED=0
# Agent/Loader 目标: Windows x64
WINDOWS_ENV=GOOS=windows GOARCH=amd64

.PHONY: all server agent loader clean help

# 默认目标：编译所有组件
all: server agent loader
	@echo "所有组件编译完成！"

# 编译 Server (Linux)
server:
	@echo "正在编译 Server (Linux)..."
	@mkdir -p $(BUILD_DIR)
	$(LINUX_ENV) $(GOBUILD) -ldflags "-s -w" -o $(BUILD_DIR)/$(SERVER_NAME) $(SERVER_MAIN)
	@echo "Server 编译完成: $(BUILD_DIR)/$(SERVER_NAME)"

# 编译 Agent (Windows)
agent:
	@echo "正在编译 Agent (Windows)..."
	@mkdir -p $(BUILD_DIR)
	$(WINDOWS_ENV) $(GOBUILD) -ldflags "-s -w" -o $(BUILD_DIR)/$(AGENT_NAME) $(AGENT_MAIN)
	@echo "Agent 编译完成: $(BUILD_DIR)/$(AGENT_NAME)"

# 编译 Loader (Windows)
loader:
	@echo "正在编译 Loader (Windows)..."
	@mkdir -p $(BUILD_DIR)
	$(WINDOWS_ENV) $(GOBUILD) -ldflags "-s -w" -o $(BUILD_DIR)/$(LOADER_NAME) $(LOADER_MAIN)
	@echo "Loader 编译完成: $(BUILD_DIR)/$(LOADER_NAME)"

# 清理编译产物
clean:
	@echo "正在清理编译产物..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	@echo "清理完成！"

# 运行 Server
run-server: server
	@echo "启动 Server..."
	cd $(BUILD_DIR) && cp -r ../web . && ./$(SERVER_NAME)

# 显示帮助信息
help:
	@echo "BasicC2 编译脚本"
	@echo ""
	@echo "用法:"
	@echo "  make all          - 编译所有组件"
	@echo "  make server       - 编译 Server"
	@echo "  make agent        - 编译 Agent (Windows)"
	@echo "  make loader       - 编译 Loader (Windows)"
	@echo "  make run-server   - 编译并运行 Server"
	@echo "  make clean        - 清理编译产物"
	@echo "  make help         - 显示此帮助信息"
