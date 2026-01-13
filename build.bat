@echo off
REM BasicC2 Windows 编译脚本
REM 用法: build.bat [all|server|agent|loader|clean]

if "%1"=="clean" goto clean
if "%1"=="server" goto server
if "%1"=="agent" goto agent
if "%1"=="loader" goto loader
if "%1"=="" goto all
if "%1"=="all" goto all

echo 未知命令: %1
echo 用法: build.bat [all^|server^|agent^|loader^|clean]
goto end

:all
echo ========================================
echo 编译所有组件
echo ========================================
call :server
call :agent
call :loader
echo.
echo ========================================
echo 编译完成！
echo ========================================
echo build/server       (Linux - 部署到 Ubuntu)
echo build/agent.exe    (Windows - 本机运行)
echo build/loader.exe   (Windows - 本机运行)
echo ========================================
goto end

:server
echo.
echo [1/3] 正在编译 Server (Linux 版本)...
if not exist build mkdir build
set GOOS=linux
set GOARCH=arm64
go build -ldflags "-s -w" -o build/server cmd/server/main.go
if errorlevel 1 (
    echo [错误] Server 编译失败
    exit /b 1
)
echo [完成] Server 编译成功: build/server
goto :eof

:agent
echo.
echo [2/3] 正在编译 Agent (Windows 版本)...
if not exist build mkdir build
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-s -w " -o build/agent.exe cmd/agent/main.go
if errorlevel 1 (
    echo [错误] Agent 编译失败
    exit /b 1
)
echo [完成] Agent 编译成功: build/agent.exe
goto :eof

:loader
echo.
echo [3/3] 正在编译 Loader (Windows 版本)...
if not exist build mkdir build
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-s -w " -o build/loader.exe cmd/loader/main.go
if errorlevel 1 (
    echo [错误] Loader 编译失败
    exit /b 1
)
echo [完成] Loader 编译成功: build/loader.exe
goto :eof

:clean
echo 正在清理编译产物...
if exist build rmdir /s /q build
go clean
echo 清理完成！
goto end

:end
