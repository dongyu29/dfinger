package network

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// HTTPClient 用于定义 HTTP 客户端的可配置选项
type HTTPClient struct {
	Timeout             time.Duration // 超时时间
	FollowRedirects     bool          // 是否跟随重定向
	MaxIdleConns        int           // 最大空闲连接数
	MaxConnsPerHost     int           // 每主机最大连接数
	MaxIdleConnsPerHost int           // 每主机最大空闲连接数
	Proxy               string        // 代理地址 (如 "http://127.0.0.1:8080")
	DisableKeepAlives   bool          // 是否禁用 Keep-Alive
	InsecureSkipVerify  bool          // 是否跳过 TLS 证书验证
	Http2               bool          //控制是否强制尝试使用 HTTP/2 协议
}

func NewDefaultHTTPClient() *http.Client {
	opts := HTTPClient{
		Timeout:             5 * time.Second,
		FollowRedirects:     true,
		MaxIdleConns:        2000,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     1000,
		Proxy:               "",
		DisableKeepAlives:   false,
		InsecureSkipVerify:  true,
		Http2:               false,
	}
	// 创建 HTTP 客户端
	return NewHTTPClient(opts)
}

// NewHTTPClient 创建一个功能丰富的自定义 HTTP 客户端
func NewHTTPClient(opts HTTPClient) *http.Client {
	// 自定义传输层
	transport := &http.Transport{
		ForceAttemptHTTP2: opts.Http2,
		Proxy: func(req *http.Request) (*url.URL, error) {
			if opts.Proxy != "" {
				return url.Parse(opts.Proxy)
			}
			return nil, nil
		},
		DialContext: (&net.Dialer{
			Timeout:   opts.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: opts.Timeout,
		MaxIdleConns:        opts.MaxIdleConns,
		MaxIdleConnsPerHost: opts.MaxIdleConnsPerHost,
		MaxConnsPerHost:     opts.MaxConnsPerHost,
		DisableKeepAlives:   opts.DisableKeepAlives,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS10,
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
	}

	client := &http.Client{
		Timeout:   opts.Timeout,
		Transport: transport,
	}

	// 配置是否跟随重定向
	if !opts.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client
}

// IsRetryable 判断是否需要重试
func IsRetryable(resp *http.Response, err error) bool {
	if err != nil {
		// 网络错误需要重试
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Temporary() {
			return true
		}
		return false
	}
	// HTTP 状态码错误（如 500~599）需要重试
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		return true
	}
	return false
}

// 处理并重置响应体
func readAndResetBody(resp *http.Response) (string, error) {
	// 读取响应体内容
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// 将响应体内容转换为字符串
	body := string(bodyBytes)

	// 恢复响应体，使其可以再次读取
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return body, nil
}

// DoWithRetry 执行带重试逻辑的 HTTP 请求
// 此处的尝试次数retryCount int尚未完成用户参数可控
func DoWithRetry(client *http.Client, req *http.Request, retryCount int, retryDelay time.Duration, JsRedirect int) (*http.Response, string, error) {
	var resp *http.Response
	var err error
	var body string

	if client == nil {
		return nil, "", fmt.Errorf("client is nil")
	}
	if req == nil {
		return nil, "", fmt.Errorf("request is nil")
	}

	for i := 0; i <= retryCount; i++ {
		resp, err = client.Do(req)
		if resp != nil {
			if !IsRetryable(resp, err) {
				body, _ = readAndResetBody(resp)
				break
			}
		}
	}

	// 如果在所有重试后仍未获得响应
	if resp == nil {
		return &http.Response{}, "", err
	}

	// 判断是否进行 JS 跳转
	if JsRedirect > 0 {
		// 处理 JS 跳转
		for i := 0; i < JsRedirect; i++ {
			jumpurl := Jsjump(resp, body)
			if jumpurl == "" {
				break
			}
			// 更新请求的 URL
			req.URL, err = url.Parse(jumpurl)
			if err != nil {
				return resp, body, fmt.Errorf("failed to parse jump URL: %v", err)
			}

			// 再次发起请求
			resp, err = client.Do(req)
			if err != nil {
				return resp, body, err
			}
			// 关闭响应体
			//defer resp.Body.Close()

			// 读取并重置新的响应体
			if resp != nil {
				body, err = readAndResetBody(resp)
			}
			if err != nil {
				return &http.Response{}, "", err
			}
		}
		return resp, body, nil
	} else {
		// 如果仍然有错误，则返回错误
		if err != nil {
			return resp, body, err
		}
		return resp, body, nil
	}
}
