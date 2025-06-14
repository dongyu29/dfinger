package finger

import (
	"bytes"
	"dfinger/common"
	"dfinger/core/network"
	"encoding/base64"
	"fmt"
	"github.com/spaolacci/murmur3"
	"hash"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func AnalyzeResponse(resp *http.Response, body string, req *http.Request, client *http.Client, urlInfo common.UrlInfo) (string, string, string, int, []DetectionResult) {
	var (
		title         string
		iconURL       string
		iconHash      string
		contentLength int
		fingers       []DetectionResult
	)

	title = ExtractTitle(strings.ReplaceAll(strconv.Itoa(resp.StatusCode), "206", "200"), resp.Header.Get("Content-Type"), body, resp.Header, 40)
	iconURL, iconHash, _ = GetFaviconHash(req, client, body, req.URL)

	// 优化 Content-Length 处理，若有指定优先使用指定的值
	if contentLen := resp.Header.Get("Content-Length"); contentLen != "" {
		contentLength, _ = strconv.Atoi(contentLen)
	} else {
		contentLength = len(body)
	}

	detector := NewDetector(Rules)
	fingers = detector.Detect(resp, []byte(body), title, iconHash, urlInfo.Path)

	return title, iconURL, iconHash, contentLength, fingers
}

func ExtractTitle(code string, ctype string, body string, headers http.Header, TitleLen int) string {

	// 如果是 3xx 重定向状态码，返回 Location 头部的 URL
	if strings.HasPrefix(code, "3") {
		return fmt.Sprintf("%s-> %s", code, headers.Get("Location"))
	}

	// 如果 Content-Type 是 JSON、纯文本或空，处理响应体并返回标题
	if strings.Contains(ctype, "json") || strings.Contains(ctype, "plain") || ctype == "" {
		title := ReplaceStrings(body, " ", "\n", "[", "]")

		// 如果标题长度超出最大长度，截断并加上 "..."
		if len(title) > TitleLen {
			return title[:TitleLen] + "..."
		}

		return title
	}

	// 如果 Content-Type 是 HTML，提取 <title> 标签中的内容
	if strings.Contains(ctype, "html") {
		// 提取 <title> 标签内容
		start := strings.Index(body, "<title>")
		end := strings.Index(body, "</title>")
		if start != -1 && end != -1 && start < end {
			title := strings.TrimSpace(body[start+len("<title>") : end])

			// 如果标题长度超出最大长度，截断并加上 "..."
			if len(title) > TitleLen {
				return title[:TitleLen] + "..."
			}
			return title
		}
	}
	return "Unknown Title"
}

func ReplaceStrings(input string, replacements ...string) string {
	// 假设我们会做多次替换，按顺序进行处理
	for i := 0; i < len(replacements); i += 2 {
		if i+1 < len(replacements) {
			input = strings.ReplaceAll(input, replacements[i], replacements[i+1])
		}
	}
	return input
}

// 提取 favicon URL 和 hash（传入 body 为 string，baseURL 为 *url.URL）
func GetFaviconHash(req *http.Request, client *http.Client, body string, baseURL *url.URL) (path string, iconHash string, err error) {
	// 多模式查找 favicon URL
	favURL, path, err := findFaviconURL(baseURL, body)
	if err != nil {
		return "", "", fmt.Errorf("parse favicon URL failed: %w", err)
	}
	iconData, err := fetchFavicon(client, req, favURL)
	if err != nil {
		return "", "", fmt.Errorf("fetch favicon failed: %w", err)
	}

	hash := Mmh3Hash32(StandBase64(iconData))
	return path, hash, nil
}

// 查找 favicon URL 的独立函数
// 优先匹配 rel="icon" 声明
// 检查 OpenGraph/Twitter 图片作为备用
// 扫描 7 个常见路径（带存在性验证）
// 最终回退到 /favicon.ico
func findFaviconURL(baseURL *url.URL, html string) (*url.URL, string, error) {
	// 正则优先匹配：<link rel="...icon..." href="...">，无论属性顺序
	patterns := []struct {
		re  *regexp.Regexp
		idx int
	}{
		// 改进后的 link rel icon 规则
		{regexp.MustCompile(`(?i)<link[^>]*?rel=["'][^"']*icon[^"']*["'][^>]*?href=["']?([^"'\s>]+)`), 1},

		// OpenGraph 图像
		{regexp.MustCompile(`(?i)<meta[^>]+property=["']og:image["'][^>]+content=["']?([^"'>]+)`), 1},

		// Twitter 图像
		{regexp.MustCompile(`(?i)<meta[^>]+name=["']twitter:image["'][^>]+content=["']?([^"'>]+)`), 1},
	}

	// 尝试从 HTML 中提取 favicon
	for _, p := range patterns {
		if matches := p.re.FindStringSubmatch(html); len(matches) > p.idx {
			path := strings.TrimSpace(matches[p.idx])
			if parsed, err := baseURL.Parse(path); err == nil {
				return parsed, path, nil
			}
		}
	}

	// 常见路径回退
	commonPaths := []string{
		"/favicon.ico", "/favicon.png", "/favicon.jpg",
		"/assets/favicon.ico", "/static/favicon.ico",
		"/img/favicon.ico", "/images/favicon.ico",
	}
	for _, path := range commonPaths {
		if parsed, err := baseURL.Parse(path); err == nil {
			if checkURLExists(parsed) {
				return parsed, path, nil
			}
		}
	}

	// 最终兜底
	final, _ := baseURL.Parse("/favicon.ico")
	return final, "/favicon.ico", nil
}

// 下载并验证 favicon
func fetchFavicon(client *http.Client, baseReq *http.Request, targetURL *url.URL) ([]byte, error) {
	req := cloneRequest(baseReq) // 重要：避免修改原始请求
	req.URL = targetURL
	req.Method = "GET"

	// 带重试的请求（建议最大 3 次）
	resp, _, err := network.DoWithRetry(client, req, 3, 1*time.Second, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 验证响应
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// 检查内容类型
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
		return nil, fmt.Errorf("invalid content type: %s", ct)
	}

	// 限制读取大小（最大 2MB）
	return io.ReadAll(resp.Body)
	//return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

// StandBase64 计算 base64 的值
func StandBase64(braw []byte) []byte {
	bckd := base64.StdEncoding.EncodeToString(braw)
	var buffer bytes.Buffer
	for i := 0; i < len(bckd); i++ {
		ch := bckd[i]
		buffer.WriteByte(ch)
		if (i+1)%76 == 0 {
			buffer.WriteByte('\n')
		}
	}
	buffer.WriteByte('\n')

	return buffer.Bytes()
}

/*
Mmh3Hash32 计算 mmh3 hash
*/
func Mmh3Hash32(raw []byte) string {
	var h32 hash.Hash32 = murmur3.New32()
	h32.Write(raw)

	return fmt.Sprintf("%d", int32(h32.Sum32()))
}

// checkURLExists 使用 HEAD 请求验证指定的 URL 是否存在
func checkURLExists(url *url.URL) bool {
	// 创建一个 HTTP 请求
	req, err := http.NewRequest("HEAD", url.String(), nil)
	if err != nil {
		return false
	}

	// 设置一些合理的请求超时时间
	client := &http.Client{
		Timeout: time.Duration(common.Infos.Timeout) * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 如果状态码为 2xx，说明 URL 存在
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// 辅助函数：复制请求
func cloneRequest(r *http.Request) *http.Request {
	clone := r.Clone(r.Context())
	clone.Body = nil // DoWithRetry 应该处理 Body
	return clone
}
