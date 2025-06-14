package common

import (
	"dfinger/core/DNS"
)

var (
	Finger_file    = "resource/fingers.json"
	Cdn_cname_file = "resource/cdn_cname.txt"
	Cdn_ip_file    = "resource/cdn_ip.txt"
	DefaultPorts   = "80,81,88,99,443,800,801,808,888,1000,1010,1080,1099,2375,2379,3000,3128,5000,5003,5555,6080,7001,7002,7070,7071,7080,7200,7777,7890,8000,8001,8008,8010,8011,8020,8028,8030,8042,8053,8069,8070,8080,8081,8083,8088,8090,8091,8096,8100,8118,8161,8180,8181,8200,8222,8244,8280,8360,8443,8484,8800,8848,8868,8880,8888,8899,8983,8989,9000,9001,9002,9008,9010,9043,9060,9080,9081,9088,9090,9091,9100,9200,9443,9800,9981,9988,9999,10000,10001,10250,12443,18000,18080,18088,19001,20000,20880"

	DnsServers = []string{
		"8.8.8.8",         // Google DNS
		"9.9.9.9",         // Quad9 DNS
		"114.114.114.114", // 114DNS
		"223.5.5.5",       // 阿里云 DNS
		"180.76.76.76",    // 百度 DNS
		"1.1.1.1",         // Cloudflare DNS (国际备选)
	}
)

var Resolver = DNS.NewDNSResolver(DnsServers)

type Info struct {
	TargetAddr string // -a 目标
	TargetFile string // -f 批量目标文件
	Ports      string // -p 端口
	OutputFile string // -o 输出结果文件
	Threads    int    // -c 并发线程数
	Timeout    int    // -t 超时时间（秒）
	FingerFile string // -finger 指纹库文件路径
}

var Infos Info
