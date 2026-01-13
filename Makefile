# Makefile for BasicC2

# 变量定义
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt

# 输出目录
BUILD_DIR=build
EXAMPLE_DIR=examples

# 服务器配置
SERVER_NAME=server
SERVER_MAIN=./cmd/server/main.go

# Agent 配置 (Windows)
AGENT_NAME=agent.exe
AGENT_MAIN=./cmd/agent/main.go

# Loader 配置 (Windows)
LOADER_NAME=loader.exe
LOADER_MAIN=./cmd/loader/main.go

# 示例程序已删除
# LATERAL_EXAMPLE=lateral-example.exe
# LATERAL_EXAMPLE_MAIN=./examples/lateral/main.go

# 交叉编译环境变量
# Server 目标: Ubuntu 22.04 ARM64
LINUX_ENV=GOOS=linux GOARCH=arm64 CGO_ENABLED=0
# Agent/Loader/Example 目标: Windows x64
WINDOWS_ENV=GOOS=windows GOARCH=amd64
# macOS 本地编译
DARWIN_ENV=GOOS=darwin GOARCH=arm64

# 编译标志
LDFLAGS=-ldflags "-s -w"
VERBOSE_LDFLAGS=-ldflags "-s -w -X main.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev') -X main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

.PHONY: all server agent loader example clean test vet fmt check run-server run-server-local help deps

# 默认目标：编译所有组件
all: server agent loader
	@echo "✅ 所有组件编译完成！"
	@echo ""
	@echo "📦 编译产物："
	@ls -lh $(BUILD_DIR)

# 编译所有组件
full: server agent loader
	@echo "✅ 所有组件编译完成！"

# 编译 Server (Linux ARM64)
server:
	@echo "🔨 正在编译 Server (Linux ARM64)..."
	@mkdir -p $(BUILD_DIR)
	$(LINUX_ENV) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_NAME) $(SERVER_MAIN)
	@echo "✅ Server 编译完成: $(BUILD_DIR)/$(SERVER_NAME)"

# 编译 Server (macOS 本地)
server-local:
	@echo "🔨 正在编译 Server (macOS ARM64)..."
	@mkdir -p $(BUILD_DIR)
	$(DARWIN_ENV) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_NAME)-mac $(SERVER_MAIN)
	@echo "✅ Server 编译完成: $(BUILD_DIR)/$(SERVER_NAME)-mac"

# 编译 Agent (Windows)
agent:
	@echo "🔨 正在编译 Agent (Windows x64)..."
	@mkdir -p $(BUILD_DIR)
	$(WINDOWS_ENV) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(AGENT_NAME) $(AGENT_MAIN)
	@echo "✅ Agent 编译完成: $(BUILD_DIR)/$(AGENT_NAME)"

# 编译 Loader (Windows)
loader:
	@echo "🔨 正在编译 Loader (Windows x64)..."
	@mkdir -p $(BUILD_DIR)
	$(WINDOWS_ENV) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(LOADER_NAME) $(LOADER_MAIN)
	@echo "✅ Loader 编译完成: $(BUILD_DIR)/$(LOADER_NAME)"

# 示例程序已删除
# example:
# 	@echo "🔨 正在编译横向移动示例 (Windows x64)..."
# 	@mkdir -p $(BUILD_DIR)
# 	$(WINDOWS_ENV) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(LATERAL_EXAMPLE) $(LATERAL_EXAMPLE_MAIN)
# 	@echo "✅ 示例程序编译完成: $(BUILD_DIR)/$(LATERAL_EXAMPLE)"

# 测试相关
test:
	@echo "🧪 运行所有单元测试..."
	$(GOTEST) -v -cover ./...

test-lateral:
	@echo "🧪 运行横向移动模块测试..."
	$(GOTEST) -v -cover ./agent/lateral/...

test-coverage:
	@echo "🧪 生成测试覆盖率报告..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告已生成: coverage.html"

# 代码检查
vet:
	@echo "🔍 运行代码静态检查..."
	$(GOVET) ./...

fmt:
	@echo "🎨 格式化代码..."
	$(GOFMT) ./...

check: fmt vet test
	@echo "✅ 代码检查完成！"

# 清理编译产物
clean:
	@echo "🧹 正在清理编译产物..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "✅ 清理完成！"

# 运行 Server（Linux 版本，需要在 Linux 环境）
run-server: server
	@echo "🚀 启动 Server (Linux)..."
	cd $(BUILD_DIR) && cp -r ../web . && ./$(SERVER_NAME)

# 运行 Server（macOS 本地版本）
run-server-local: server-local
	@echo "🚀 启动 Server (macOS)..."
	cd $(BUILD_DIR) && cp -r ../web . && ./$(SERVER_NAME)-mac

# 开发模式：直接运行不编译
dev-server:
	@echo "🚀 开发模式启动 Server..."
	@mkdir -p $(BUILD_DIR)
	@cp -r ./web $(BUILD_DIR)/
	cd $(BUILD_DIR) && $(GOCMD) run ../$(SERVER_MAIN)

# 安装依赖
deps:
	@echo "📦 安装项目依赖..."
	$(GOCMD) mod download
	$(GOCMD) mod tidy
	@echo "✅ 依赖安装完成！"

# 显示项目信息
info:
	@echo "📊 项目信息"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "项目名称: BasicC2"
	@echo "Go 版本: $(shell $(GOCMD) version)"
	@echo "模块路径: $(shell head -1 go.mod | cut -d' ' -f2)"
	@echo "构建目录: $(BUILD_DIR)"
	@echo ""
	@echo "📁 文件统计"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "Go 文件: $(shell find . -name '*.go' -not -path './vendor/*' | wc -l | xargs)"
	@echo "代码行数: $(shell find . -name '*.go' -not -path './vendor/*' | xargs wc -l | tail -1 | awk '{print $$1}')"

# 显示帮助信息
help:
	@echo "🔧 BasicC2 编译脚本"
	@echo ""
	@echo "📦 编译命令:"
	@echo "  make all              - 编译所有核心组件 (server, agent, loader)"
	@echo "  make full             - 编译所有组件（包括示例）"
	@echo "  make server           - 编译 Server (Linux ARM64)"
	@echo "  make server-local     - 编译 Server (macOS ARM64)"
	@echo "  make agent            - 编译 Agent (Windows x64)"
	@echo "  make loader           - 编译 Loader (Windows x64)"
	@echo ""
	@echo "🧪 测试命令:"
	@echo "  make test             - 运行所有单元测试"
	@echo "  make test-lateral     - 运行横向移动模块测试"
	@echo "  make test-coverage    - 生成测试覆盖率报告"
	@echo ""
	@echo "🔍 代码质量:"
	@echo "  make vet              - 运行代码静态检查"
	@echo "  make fmt              - 格式化代码"
	@echo "  make check            - 运行完整检查 (fmt + vet + test)"
	@echo ""
	@echo "🚀 运行命令:"
	@echo "  make run-server       - 编译并运行 Server (Linux)"
	@echo "  make run-server-local - 编译并运行 Server (macOS)"
	@echo "  make dev-server       - 开发模式启动 Server (不编译)"
	@echo ""
	@echo "🛠️  工具命令:"
	@echo "  make deps             - 安装/更新项目依赖"
	@echo "  make clean            - 清理编译产物"
	@echo "  make info             - 显示项目信息"
	@echo "  make help             - 显示此帮助信息"
	@echo ""
	@echo "💡 快速开始:"
	@echo "  1. make deps          # 安装依赖"
	@echo "  2. make all           # 编译所有组件"
	@echo "  3. make run-server-local  # 启动服务器 (macOS)"
	@echo "  4. 访问 http://localhost:8080"
