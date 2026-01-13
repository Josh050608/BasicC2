package commands

import (
	"bytes"
	"io"
	"os/exec"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// Execute 执行命令并返回结果
// 支持特殊命令：screenshot
func Execute(cmdStr string) string {
	// 检查是否是截图命令
	if strings.TrimSpace(cmdStr) == "screenshot" {
		return TakeScreenshot()
	}

	// 执行普通的 CMD 命令
	cmd := exec.Command("cmd", "/C", cmdStr)
	output, _ := cmd.CombinedOutput()
	
	// 处理中文编码问题
	utf8Output, err := gbkToUtf8(output)
	if err != nil {
		utf8Output = string(output)
	}
	
	return utf8Output
}

// gbkToUtf8 将 GBK 编码转换为 UTF-8
func gbkToUtf8(s []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}
