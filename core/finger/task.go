package finger

import (
	"dfinger/common"
	"dfinger/core/network"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/projectdiscovery/gologger"
	"net/http"
	"sync"
	"time"
)

type ScanTask struct {
	Req     *http.Request
	UrlInfo common.UrlInfo
	Cdninfo *network.CDNInfo
}

func RunTask(urls []common.UrlInfo, client *http.Client) {
	if client == nil {
		gologger.Info().Msgf("HTTP client is nil.")
		return
	}

	var wg sync.WaitGroup
	taskChan := make(chan ScanTask, len(urls)*5) // 预估放大容量
	numWorkers := common.Infos.Threads

	// 启动工作协程池
	for i := 0; i < numWorkers; i++ {
		go worker(taskChan, client, &wg)
	}

	tasks := GenerateScanTasks(urls)
	for _, task := range tasks {
		wg.Add(1)
		taskChan <- task
	}

	close(taskChan)
	wg.Wait()
}

func worker(taskChan chan ScanTask, client *http.Client, wg *sync.WaitGroup) {
	defer func() {
		if r := recover(); r != nil {
			gologger.Error().Msgf("任务崩溃: %v", r)
		}
	}()
	if client == nil {
		gologger.Info().Msgf("HTTP client is nil.")
		return
	}

	for task := range taskChan {
		sendRequest(task.Req, client, task.UrlInfo, wg, *task.Cdninfo)
	}
}

// sendRequest 发送 HTTP 请求并处理响应
func sendRequest(req *http.Request, client *http.Client, urlInfo common.UrlInfo, wg *sync.WaitGroup, cdninfo network.CDNInfo) {
	var (
		iconHash      string
		title         string
		body          string
		contentLength int
		fingers       []DetectionResult
	)

	if client == nil {
		gologger.Info().Msgf("HTTP client is nil.")
		wg.Done()
		return
	}

	if urlInfo.Host == "" || urlInfo.Scheme == "" || urlInfo.Port == "" {
		gologger.Info().Msgf("Invalid UrlInfo: %+v", urlInfo)
		wg.Done()
		return
	}

	// 使用带重试的请求发送器
	resp, body, err := network.DoWithRetry(client, req, 2, 1*time.Second, 3)
	if err != nil {
		gologger.Debug().Msgf("请求失败: %v\n", err)
		wg.Done()
		return
	} else {
		gologger.Debug().Msgf("%v请求结束", req.URL.String())
	}
	defer resp.Body.Close()

	// 分析返回数据
	if resp != nil && resp.Body != nil {
		title, _, iconHash, contentLength, fingers = AnalyzeResponse(resp, body, req, client, urlInfo)

		PrintResult(urlInfo.Scheme+"://"+urlInfo.Host+":"+urlInfo.Port+urlInfo.Path, resp.StatusCode, title, contentLength, iconHash, fingers)

	}

	wg.Done()
}

func GenerateScanTasks(urls []common.UrlInfo) []ScanTask {
	var tasks []ScanTask
	for _, urlInfo := range urls {
		cdnInfo := network.NewCDNInfo()

		if urlInfo.IsDomain {
			if network.DefaultCDNChecker.IsCDNCNAME(urlInfo.Host) {
				gologger.Info().Msgf(aurora.Red(fmt.Sprintf("%v 命中CDN CNAME", urlInfo.Host)).String())
				cdnInfo.MarkAsCDN()
			}

			resolver := common.Resolver
			ips, err := resolver.LookupIP(urlInfo.Host)
			if err != nil {
				gologger.Info().Msgf("DNS 解析失败 (%s): %v\n", urlInfo.Host, err)
				continue
			}

			for _, ip := range ips {
				url := fmt.Sprintf("%s://%s:%s%s", urlInfo.Scheme, ip.String(), urlInfo.Port, urlInfo.Path)
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					gologger.Debug().Msgf("构造请求失败: %v", err)
					continue
				}
				req.Host = urlInfo.Host

				if network.DefaultCDNChecker.IsCDNIP(ip) {
					gologger.Info().Msgf(aurora.Red(fmt.Sprintf("%v 命中CDN IP段", string(ip))).String())
					cdnInfo.MarkAsCDN()
					cdnInfo.AddCDNIP(ip)
				} else {
					cdnInfo.AddRealIP(ip)
				}

				tasks = append(tasks, ScanTask{Req: req, UrlInfo: urlInfo, Cdninfo: cdnInfo})
			}
		} else {
			url := fmt.Sprintf("%s://%s:%s%s", urlInfo.Scheme, urlInfo.Host, urlInfo.Port, urlInfo.Path)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				gologger.Debug().Msgf("构造请求失败: %v", err)
				continue
			}
			tasks = append(tasks, ScanTask{Req: req, UrlInfo: urlInfo, Cdninfo: cdnInfo})
		}
	}
	return tasks
}
