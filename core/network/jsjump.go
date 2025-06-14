package network

import (
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reg1    = regexp.MustCompile(`(?i)<meta.*?http-equiv=.*?refresh.*?url=(.*?)/?>`)
	reg2    = regexp.MustCompile(`(?i)[window\.]?location[\.href]?.*?=.*?["'](.*?)["']`)
	reg3    = regexp.MustCompile(`(?i)[window\.]?location\.replace\(['"](.*?)['"]\)`)
	regHost = regexp.MustCompile(`(?i)https?://(.*?)/`)
)

func Jsjump(resp *http.Response, body string) string {
	res := regexJsjump(body)
	if res != "" && res != "http:" {
		res = strings.TrimSpace(res)
		res = strings.ReplaceAll(res, "\"", "")
		res = strings.ReplaceAll(res, "'", "")
		res = strings.ReplaceAll(res, "./", "/")
		if strings.HasPrefix(res, "http") {
			matches := regHost.FindAllStringSubmatch(res, -1)
			if len(matches) > 0 {
				var ip net.IP
				if strings.Contains(matches[0][1], ":") {
					ip = net.ParseIP(strings.Split(matches[0][1], ":")[0])
				} else {
					ip = net.ParseIP(matches[0][1])
				}
				if HasLocalIP(ip) {
					baseUrl := resp.Request.Host
					res = strings.ReplaceAll(res, matches[0][1], baseUrl)
				}
			}
			return res
		} else if strings.HasPrefix(res, "/") {
			baseUrl := resp.Request.URL.Scheme + "://" + resp.Request.Host
			return baseUrl + res
		} else {
			baseUrl := resp.Request.URL.Scheme + "://" + resp.Request.Host + "/" + filepath.Dir(resp.Request.URL.Path) + "/"
			baseUrl = strings.ReplaceAll(baseUrl, "./", "")
			baseUrl = strings.ReplaceAll(baseUrl, "///", "/")
			return baseUrl + res
		}
	}
	return ""
}

func regexJsjump(body string) string {
	matches := reg1.FindAllStringSubmatch(body, -1)
	if len(matches) > 0 {
		if !strings.Contains(body, "<!--\r\n"+matches[0][0]) && !strings.Contains(matches[0][1], "nojavascript.html") && !strings.Contains(body, "<!--[if lt IE 7]>\n"+matches[0][0]) {
			return matches[0][1]
		}
	}
	if len(body) > 700 {
		body = body[:700]
	}
	matches = reg2.FindAllStringSubmatch(body, -1)
	if len(matches) > 0 {
		return matches[0][1]
	}
	matches = reg3.FindAllStringSubmatch(body, -1)
	if len(matches) > 0 {
		return matches[0][1]
	}
	return ""
}

func HasLocalIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0] == 10 ||
		(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
		(ip4[0] == 169 && ip4[1] == 254) ||
		(ip4[0] == 192 && ip4[1] == 168)
}
