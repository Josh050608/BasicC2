//经测试截图功能有BUG

package commands

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"

	"github.com/kbinani/screenshot"
)

// TakeScreenshot 截取屏幕并返回 Base64 编码的 PNG 图片
func TakeScreenshot() string {
	// 获取显示器数量
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return "Error: No display found"
	}

	// 截取第一个屏幕
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
