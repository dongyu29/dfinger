package finger

import (
	"dfinger/common"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/projectdiscovery/gologger"
	"os"
	"strings"
)

func PrintResult(
	host string,
	statusCode int,
	title string,
	contentLength int,
	iconHash string,
	fingers []DetectionResult,
) {
	// Host 蓝色
	hostColored := aurora.BrightBlue(host).String()

	// 状态码颜色
	var statusColored string
	switch {
	case statusCode == 200:
		statusColored = aurora.Green(fmt.Sprintf("%d", statusCode)).String()
	case statusCode >= 300 && statusCode < 400:
		statusColored = aurora.Yellow(fmt.Sprintf("%d", statusCode)).String()
	case statusCode >= 400:
		statusColored = aurora.Red(fmt.Sprintf("%d", statusCode)).String()
	default:
		statusColored = aurora.BrightBlack(fmt.Sprintf("%d", statusCode)).String()
	}

	// Title 青色
	titleColored := aurora.Cyan(title).String()

	// contentLength 紫色
	lengthColored := aurora.Magenta(fmt.Sprintf("%d", contentLength)).String()

	// iconHash 灰色
	iconHashColored := aurora.Gray(12, iconHash).String()

	// fingers 按Level分颜色
	var fingerStrs []string
	for _, f := range fingers {
		var fingerColored string
		switch f.Level {
		case 3:
			fingerColored = aurora.Red(fmt.Sprintf("%s(L%d)", f.CMS, f.Level)).String()
		case 2:
			fingerColored = aurora.Yellow(fmt.Sprintf("%s(L%d)", f.CMS, f.Level)).String()
		default:
			fingerColored = aurora.Green(fmt.Sprintf("%s(L%d)", f.CMS, f.Level)).String()
		}
		fingerStrs = append(fingerStrs, fingerColored)
	}

	gologger.Info().Msgf(
		"%s | %s | %s | [len:%s] | iconHash: %s | Finger: %s",
		hostColored,
		statusColored,
		titleColored,
		lengthColored,
		iconHashColored,
		strings.Join(fingerStrs, ", "),
	)

	// 保存纯文本结果
	plainFingerStrs := make([]string, len(fingers))
	for i, f := range fingers {
		plainFingerStrs[i] = fmt.Sprintf("%s(L%d)", f.CMS, f.Level)
	}

	plainOutput := fmt.Sprintf("[+] %s | %d | %s | [len:%d] | iconHash: %s | Finger: %s\n",
		host, statusCode, title, contentLength, iconHash,
		strings.Join(plainFingerStrs, ", "))

	if common.Infos.OutputFile != "" {
		f, err := os.OpenFile(common.Infos.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			gologger.Error().Msgf("无法写入输出文件: %s", err)
			return
		}
		defer f.Close()
		_, _ = f.WriteString(plainOutput)
	}
}
