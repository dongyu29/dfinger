package finger

import (
	"encoding/json"
	"github.com/projectdiscovery/gologger"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

// Condition 单个匹配条件
type Condition struct {
	Location string   `json:"location"` // header, body, title, favicon, path
	Matcher  string   `json:"matcher"`  // match, regex  支持关键字匹配、正则匹配
	Keywords []string `json:"keywords"` // 关键字列表
}

// FingerprintRule 指纹规则
type FingerprintRule struct {
	CMS        string      `json:"cms"`        // CMS名
	Level      int         `json:"level"`      // 置信度 1-5
	Logic      string      `json:"logic"`      // "and" 或 "or"
	Tags       []string    `json:"tags"`       // 标签
	Conditions []Condition `json:"conditions"` // 条件数组
}

var Rules []FingerprintRule

// DetectionResult 检测结果
type DetectionResult struct {
	CMS     string
	Level   int
	Tags    []string
	Matched []string // 匹配上的关键词
}

// FingerprintDetector 检测器
type FingerprintDetector struct {
	rules []FingerprintRule
	mu    sync.RWMutex
}

func NewDetector(rules []FingerprintRule) *FingerprintDetector {
	return &FingerprintDetector{
		rules: rules,
	}
}

// matchCondition 判断单个条件是否满足，返回匹配上的关键词列表
// matchCondition 判断单个条件是否满足，返回匹配上的关键词列表
func matchCondition(cond Condition, resp *http.Response, body string, title string, faviconHash string, path string) (matched []string, ok bool) {
	var data string

	switch cond.Location {
	case "header":
		// 合并所有响应头字段内容
		var headers []string
		for k := range resp.Header {
			headers = append(headers, k+": "+strings.Join(resp.Header.Values(k), ","))
		}
		data = strings.Join(headers, "\n")

	case "body":
		data = body

	case "title":
		data = title

	case "favicon":
		data = faviconHash
	case "path":
		data = path

	default:
		return nil, false
	}

	allMatch := true
	for _, keyword := range cond.Keywords {
		switch cond.Matcher {
		case "match":
			if !strings.Contains(strings.ToLower(data), strings.ToLower(keyword)) {
				allMatch = false
				break
			}
			gologger.Debug().Msgf("匹配到指纹关键字（match）：%v", keyword)
			matched = append(matched, keyword)

		case "regex":
			match, _ := regexp.MatchString("(?i)"+keyword, data) // (?i) 表示不区分大小写
			if !match {
				allMatch = false
				break
			}
			gologger.Debug().Msgf("匹配到指纹关键字（regex）：%v", keyword)
			matched = append(matched, keyword)

		default:
			return nil, false
		}
	}

	if !allMatch {
		return nil, false
	}

	if len(matched) == 0 {
		return nil, false
	}
	return matched, true
}

// Detect 对单个响应进行指纹检测
func (fd *FingerprintDetector) Detect(resp *http.Response, body []byte, title string, faviconHash string, path string) []DetectionResult {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	bodyStr := string(body)
	var results []DetectionResult

	for _, rule := range fd.rules {

		allMatched := (rule.Logic == "and")
		var matchedKeywords []string

		for _, cond := range rule.Conditions {
			matched, ok := matchCondition(cond, resp, bodyStr, title, faviconHash, path)

			if rule.Logic == "and" {
				if !ok {
					allMatched = false
					break
				} else {
					matchedKeywords = append(matchedKeywords, matched...)
				}
			} else if rule.Logic == "or" {
				if ok {
					allMatched = true
					matchedKeywords = append(matchedKeywords, matched...)
					break
				} else {
					allMatched = false
				}
			} else {
				// 默认and逻辑
				if !ok {
					allMatched = false
					break
				} else {
					matchedKeywords = append(matchedKeywords, matched...)
				}
			}
		}

		if allMatched {
			results = append(results, DetectionResult{
				CMS:     rule.CMS,
				Level:   rule.Level,
				Tags:    rule.Tags,
				Matched: matchedKeywords,
			})
		}
	}

	return results
}

// LoadFingerprints 从文件加载规则
func LoadFingerprints(path string) ([]FingerprintRule, error) {
	var fps []FingerprintRule
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &fps)
	return fps, err
}
