package common

import (
	"flag"
	"fmt"
	"os"
)

func ParseFlags() {
	flag.StringVar(&Infos.TargetAddr, "a", "", "单个目标URL，如 http://example.com")
	flag.StringVar(&Infos.Ports, "p", "", "端口号")
	flag.StringVar(&Infos.TargetFile, "f", "", "目标列表文件，每行一个URL")
	flag.StringVar(&Infos.OutputFile, "o", "result.txt", "结果输出文件路径（默认 result.txt）")
	flag.IntVar(&Infos.Threads, "t", 500, "并发线程数（默认 10）")
	flag.IntVar(&Infos.Timeout, "T", 5, "请求超时时间，单位秒（默认 10）")
	flag.StringVar(&Infos.FingerFile, "finger", Finger_file, "指纹规则文件路径（默认 fingers.json）")

	flag.Usage = func() {
		fmt.Println("用法:")
		fmt.Println("  - 扫描单个目标: ./dscan-new -a http://example.com")
		fmt.Println("  - 批量扫描文件: ./dscan-new -f targets.txt")
		fmt.Println("参数:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// 参数校验
	if Infos.TargetAddr == "" && Infos.TargetFile == "" {
		fmt.Println("[!] 必须使用 -u (单个URL) 或 -f (目标文件) 参数之一")
		flag.Usage()
		os.Exit(1)
	}

	if Infos.TargetAddr != "" && Infos.TargetFile != "" {
		fmt.Println("[!] 参数冲突：-u 和 -f 不能同时使用")
		flag.Usage()
		os.Exit(1)
	}
}
