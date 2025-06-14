package network

import (
	"bufio"
	"dfinger/common"
	"log"
	"net"
	"os"
	"strings"
)

var DefaultCDNChecker *CDNChecker

type CDNChecker struct {
	cnameKeywords []string
	cdnCIDRs      []*net.IPNet
}

// 加载 CNAME 特征数据
func loadCnameList(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			result = append(result, line)
		}
	}
	return result, scanner.Err()
}

// 加载 CDN IP 段（CIDR）
func loadCDNIPList(filename string) ([]*net.IPNet, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ipNets []*net.IPNet
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		_, ipnet, err := net.ParseCIDR(line)
		if err == nil {
			ipNets = append(ipNets, ipnet)
		}
	}
	return ipNets, scanner.Err()
}

// 初始化 CDN 检查器
func NewCDNChecker(cnameFile, ipFile string) (*CDNChecker, error) {
	cnames, err := loadCnameList(cnameFile)
	if err != nil {
		return nil, err
	}
	ipnets, err := loadCDNIPList(ipFile)
	if err != nil {
		return nil, err
	}
	return &CDNChecker{
		cnameKeywords: cnames,
		cdnCIDRs:      ipnets,
	}, nil
}

// 检查某个 CNAME 是否命中 CDN 特征
func (c *CDNChecker) IsCDNCNAME(cname string) bool {
	cname = strings.ToLower(cname)
	for _, keyword := range c.cnameKeywords {
		if strings.Contains(cname, keyword) {
			return true
		}
	}
	return false
}

// 检查某个 IP 是否在 CDN IP 段内
func (c *CDNChecker) IsCDNIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, ipnet := range c.cdnCIDRs {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

func init() {
	var err error
	DefaultCDNChecker, err = NewCDNChecker(common.Cdn_cname_file, common.Cdn_ip_file)
	if err != nil {
		log.Fatal(err)
	}
}
