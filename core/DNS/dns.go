package DNS

import (
	"fmt"
	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
	"math/rand"
	"net"
	"sync"
	"time"
)

//1. 检查缓存：
//- 如果域名有缓存，直接返回缓存内容。
//2. 使用 `net.LookupIP` 尝试解析：
//- 如果解析成功：
//- 将结果存入缓存，返回结果。
//- 如果解析失败：
//- 进入备用方案。
//3. 随机选择一个自定义的 DNS 服务器。
//4. 使用 `miekg/dns` 进行查询：
//- 如果解析成功：
//- 将结果存入缓存，返回结果。
//- 如果解析失败：
//- 返回错误信息。

var dnsCache = cache.New(5*time.Minute, 10*time.Minute) // 缓存
var cacheMutex sync.RWMutex

type DNSResolver struct {
	Servers []string
}

func NewDNSResolver(servers []string) *DNSResolver {

	if len(servers) == 0 {
		servers = []string{"8.8.8.8"}
	}
	return &DNSResolver{Servers: servers}
}

func (r *DNSResolver) LookupIP(domain string) ([]net.IP, error) {
	//fmt.Println("[DEBUG] 进入 LookupIP", domain)
	//defer fmt.Println("[DEBUG] 离开 LookupIP", domain)
	if cached, found := dnsCache.Get(domain); found {
		return cached.([]net.IP), nil
	}

	ips, err := r.lookupIPWithCustomDNS(domain)
	if err == nil {
		dnsCache.Set(domain, ips, cache.DefaultExpiration)
		return ips, nil
	}

	ips, err = net.LookupIP(domain)
	if err == nil {
		dnsCache.Set(domain, ips, cache.DefaultExpiration)
		return ips, nil
	}

	return nil, fmt.Errorf("DNS解析失败: %v", err)
}

func (r *DNSResolver) lookupIPWithCustomDNS(domain string) ([]net.IP, error) {
	server := r.getRandomServer()

	client := new(dns.Client)
	message := new(dns.Msg)
	message.SetQuestion(domain+".", dns.TypeANY)
	message.RecursionDesired = true

	start := time.Now()
	resp, _, err := client.Exchange(message, server+":53")
	elapsed := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("DNS 查询失败 (%s): %v (耗时: %v)", server, err, elapsed)
	}

	var ips []net.IP
	for _, ans := range resp.Answer {
		switch t := ans.(type) {
		case *dns.A:
			ips = append(ips, t.A)
		case *dns.AAAA:
			ips = append(ips, t.AAAA)
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("没有解析到有效的 IP 地址")
	}

	dnsCache.Set(domain, ips, cache.DefaultExpiration)
	return ips, nil
}

func (r *DNSResolver) getRandomServer() string {
	rand.Seed(time.Now().UnixNano())
	return r.Servers[rand.Intn(len(r.Servers))]
}
