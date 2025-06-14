package finger

import (
	"dfinger/common"
	"dfinger/core/network"
	"github.com/projectdiscovery/gologger"
	"net"
	"net/http"
	"strconv"
)

func GenerateWebscanTasks(ip []net.IP, port []int) (output []common.UrlInfo) {
	var temp []common.UrlInfo
	temp = append(temp, common.ParseInfo.UrlInfos...)

	for _, addr := range ip {
		for _, p := range port {
			// 根据端口选择协议
			scheme := ""
			if p == 80 {
				scheme = "http"
			} else if p == 443 {
				scheme = "https"
			} else {
				// 对于其他端口，可以同时生成 http 和 https 协议
				temp = append(temp, common.UrlInfo{
					Scheme:   "http",
					Host:     addr.String(),
					Port:     strconv.Itoa(p),
					Path:     "",
					IsDomain: false,
				})

				temp = append(temp, common.UrlInfo{
					Scheme:   "https",
					Host:     addr.String(),
					Port:     strconv.Itoa(p),
					Path:     "",
					IsDomain: false,
				})
				continue
			}

			// 生成对应协议的 URL
			temp = append(temp, common.UrlInfo{
				Scheme:   scheme,
				Host:     addr.String(),
				Port:     strconv.Itoa(p),
				Path:     "",
				IsDomain: false,
			})
		}
	}

	return temp
}

func Run(input []common.UrlInfo, client *http.Client) error {
	//采用tcp探活
	gologger.Info().Msgf("探活开始")
	//探活，直接覆盖在webscan.input
	input = network.CheckAlive(input)

	gologger.Info().Msgf("探活结束，指纹识别开始")
	if input == nil {
		gologger.Info().Msgf("无目标存活")
		return nil
	}

	//执行任务，入参有 1、输入的任务  2、client对象  3、扫描选项，实现扫描功能的拓展
	RunTask(input, client)
	return nil
}
