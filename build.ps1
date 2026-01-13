# BasicC2 PowerShell Build Script
# Usage: .\build.ps1 [all|server|agent|loader|clean]

param(
    [Parameter(Position=0)]
    [ValidateSet('all','server','agent','loader','clean')]
    [string]$Target = 'all'
)

function Build-Server {
    Write-Host ""
    Write-Host "[1/3] Building Server (Linux version)..." -ForegroundColor Cyan
    if (-not (Test-Path "build")) {
        New-Item -ItemType Directory -Path "build" | Out-Null
    }
    
    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    $env:CGO_ENABLED = "0"
    
    go build -ldflags "-s -w" -o build/server cmd/server/main.go
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[Done] Server built successfully: build/server" -ForegroundColor Green
    } else {
        Write-Host "[Error] Server build failed" -ForegroundColor Red
        exit 1
    }
}

function Build-Agent {
    Write-Host ""
    Write-Host "[2/3] Building Agent (Windows version)..." -ForegroundColor Cyan
    if (-not (Test-Path "build")) {
        New-Item -ItemType Directory -Path "build" | Out-Null
    }
    
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    
    go build -ldflags "-s -w -H=windowsgui" -o build/agent.exe cmd/agent/main.go
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[Done] Agent built successfully: build/agent.exe" -ForegroundColor Green
    } else {
        Write-Host "[Error] Agent build failed" -ForegroundColor Red
        exit 1
    }
}

function Build-Loader {
    Write-Host ""
    Write-Host "[3/3] Building Loader (Windows version)..." -ForegroundColor Cyan
    if (-not (Test-Path "build")) {
        New-Item -ItemType Directory -Path "build" | Out-Null
    }
    
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    
    go build -ldflags "-s -w -H=windowsgui" -o build/loader.exe cmd/loader/main.go
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[Done] Loader built successfully: build/loader.exe" -ForegroundColor Green
    } else {
        Write-Host "[Error] Loader build failed" -ForegroundColor Red
        exit 1
    }
}

function Clean-Build {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Yellow
    if (Test-Path "build") {
        Remove-Item -Recurse -Force "build"
    }
    go clean
    Write-Host "Clean complete!" -ForegroundColor Green
}

# Main logic
switch ($Target) {
    'clean' {
        Clean-Build
    }
    'server' {
        Build-Server
    }
    'agent' {
        Build-Agent
    }
    'loader' {
        Build-Loader
    }
    'all' {
        Write-Host "========================================" -ForegroundColor Yellow
        Write-Host "Building all components" -ForegroundColor Yellow
        Write-Host "========================================" -ForegroundColor Yellow
        
        Build-Server
        Build-Agent
        Build-Loader
        
        Write-Host ""
        Write-Host "========================================" -ForegroundColor Green
        Write-Host "Build complete!" -ForegroundColor Green
        Write-Host "========================================" -ForegroundColor Green
        Write-Host "build/server       (Linux - deploy to Ubuntu)" -ForegroundColor White
        Write-Host "build/agent.exe    (Windows - local)" -ForegroundColor White
        Write-Host "build/loader.exe   (Windows - local)" -ForegroundColor White
        Write-Host "========================================" -ForegroundColor Green
    }
}
