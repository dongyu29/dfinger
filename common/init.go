package common

import (
	"fmt"
)

func Dfinger_init() {
	ParseFlags()

	// 示例输出配置内容
	fmt.Printf("[*] 扫描配置:\n")
	fmt.Printf("    单个目标: %s\n", Infos.TargetAddr)
	fmt.Printf("    目标文件: %s\n", Infos.TargetFile)
	fmt.Printf("    输出文件: %s\n", Infos.OutputFile)
	fmt.Printf("    并发数:   %d\n", Infos.Threads)
	fmt.Printf("    超时:     %d 秒\n", Infos.Timeout)
	fmt.Printf("    指纹库:   %s\n", Infos.FingerFile)

	Parse()

}
