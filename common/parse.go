package common

import (
	"bufio"
	"fmt"
	"github.com/malfunkt/iprange"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type UrlInfo struct {
	Scheme   string
	Host     string
	Port     string
	Path     string
	IsDomain bool
}

// 初步解析数据
type Parsed struct {
	Iplist   []net.IP
	UrlInfos []UrlInfo // 存放解析后的 URL 信息
	Portlist []int
}

var ParseInfo Parsed

func Parse() error {
	addr := Infos.TargetAddr
	addrfile := Infos.TargetFile
	ports := Infos.Ports

	//解析端口
	if ports != "" { //两种端口列表，一种偏向主机，一种偏向web，不同的模块会选择调用
		parsedPorts, _ := GetPorts(ports)
		ParseInfo.Portlist = parsedPorts

	} else {
		//如果没有指定端口，就采用默认端口，并且在GlobalContext.Public.Options打一个标记
		ParseInfo.Portlist, _ = GetPorts(DefaultPorts)
	}

	//解析主机
	if addr != "" {
		addresses := strings.Split(addr, ",")
		for _, address := range addresses {
			if err := ParseAddr(address); err != nil {
				return fmt.Errorf("failed to parse addr: %v", err)
			}
		}
	}

	if addrfile != "" {
		if err := ParseAddrFile(addrfile); err != nil {
			return fmt.Errorf("failed to parse addrfile: %v", err)
		}
	}

	return nil
}

func ParseAddr(addr string) error {
	// 尝试解析为 IP 列表
	parsedList, err := iprange.ParseList(addr)
	if err == nil {
		ParseInfo.Iplist = append(ParseInfo.Iplist, parsedList.Expand()...)
		return nil
	}

	// 如果解析为 IP 失败，尝试将其作为域名处理
	err = ParseUrl(addr)
	if err != nil {
		return fmt.Errorf("failed to parse domain: %v", err)
	}
	//GlobalContext.Parsed.UrlInfos = append(GlobalContext.Parsed.UrlInfos, urlInfo...)
	return nil

}

func ParseAddrFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // 跳过空行
		}

		if err := ParseAddr(line); err != nil {
			return fmt.Errorf("failed to parse line: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}
	return nil
}

// ParseUrl 解析 URL。如果未指定协议，返回两份结果（http 和 https）
// 支持解析-a https://www.test.com:1234/test/path这种格式，这种解析以后直接加入webscan任务
func ParseUrl(addr string) error {
	// 解析地址切片
	addresses := strings.Split(addr, ",")
	for _, address := range addresses {
		address = strings.TrimSpace(address) // 去除多余空格

		// 如果没有 Scheme，生成两种 URL（http 和 https）
		if !strings.Contains(address, "://") {
			if err := parseWithSchemeAndPorts(address, "http"); err != nil {
				return err
			}
			if err := parseWithSchemeAndPorts(address, "https"); err != nil {
				return err
			}
		} else {
			// 如果已指定协议，直接解析
			if err := parseWithSchemeAndPorts(address, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

// parseWithSchemeAndPorts 根据指定的协议和端口解析 URL
func parseWithSchemeAndPorts(addr, scheme string) error {
	if scheme != "" {
		addr = scheme + "://" + addr
	}

	parsedUrl, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", addr)
	}

	// 判断是否为域名（不是IP则认为是域名）
	isDomain := !isIPAddress(parsedUrl.Hostname())
	if isDomain { //此处对域名提前进行解析

		Resolver.LookupIP(parsedUrl.Hostname())

	}

	// 如果没有显式端口，根据 WebPortlist 生成多个 URLInfo
	if parsedUrl.Port() == "" {
		if parsedUrl.Path == "" {
			for _, port := range ParseInfo.Portlist {
				// 对 80 和 443 端口进行协议限制
				if (port == 80 && parsedUrl.Scheme == "https") || (port == 443 && parsedUrl.Scheme == "http") {
					continue // 跳过不合法的协议和端口组合
				}

				urlInfo := UrlInfo{
					Scheme:   parsedUrl.Scheme,
					Host:     parsedUrl.Hostname(),
					Port:     fmt.Sprintf("%d", port),
					Path:     parsedUrl.Path,
					IsDomain: isDomain,
				}
				ParseInfo.UrlInfos = append(ParseInfo.UrlInfos, urlInfo)
			}
		} else {
			port := "80"
			if parsedUrl.Scheme == "https" {
				port = "443"
			}
			urlInfo := UrlInfo{
				Scheme:   parsedUrl.Scheme,
				Host:     parsedUrl.Hostname(),
				Port:     port,
				Path:     parsedUrl.Path,
				IsDomain: isDomain,
			}
			ParseInfo.UrlInfos = append(ParseInfo.UrlInfos, urlInfo)
		}
	} else {
		// 如果显式指定了端口，直接创建 URLInfo
		port, _ := strconv.Atoi(parsedUrl.Port())
		// 对 80 和 443 端口进行协议限制
		if (port == 80 && parsedUrl.Scheme == "https") || (port == 443 && parsedUrl.Scheme == "http") {
			return nil // 跳过不合法的协议和端口组合
		}

		urlInfo := UrlInfo{
			Scheme:   parsedUrl.Scheme,
			Host:     parsedUrl.Hostname(),
			Port:     parsedUrl.Port(),
			Path:     parsedUrl.Path,
			IsDomain: isDomain,
		}
		ParseInfo.UrlInfos = append(ParseInfo.UrlInfos, urlInfo)
	}
	return nil
}

// 判断是否是合法的IP地址
func isIPAddress(host string) bool {
	return net.ParseIP(host) != nil
}

func GetPorts(portsStr string) ([]int, error) {
	var ports []int
	portParts := strings.FieldsFunc(portsStr, func(r rune) bool {
		return r == ',' || r == ' ' // 以逗号和空格为分隔符
	})

	for _, part := range portParts {
		// 检查是否为范围格式
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("无效的端口范围: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("无效的起始端口: %s", rangeParts[0])
			}
			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("无效的结束端口: %s", rangeParts[1])
			}

			for i := start; i <= end; i++ {
				if i >= 1 && i <= 65535 {
					ports = append(ports, i)
				}
			}
		} else {
			// 处理单个端口
			port, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil {
				return nil, fmt.Errorf("无效的端口: %s", part)
			}
			if port >= 1 && port <= 65535 {
				ports = append(ports, port)
			} else {
				return nil, fmt.Errorf("端口超出范围: %d", port)
			}
		}
	}
	return ports, nil

}
