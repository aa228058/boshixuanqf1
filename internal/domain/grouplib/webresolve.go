package grouplib

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ogTitleRe / htmlTitleRe 从 t.me 页面提取标题。邀请链接页在群未隐藏预览时,
// og:title / <title> 里就是群名(形如 "群名"); 隐藏预览的群拿不到, 按失败处理。
var (
	ogTitleRe   = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']`)
	htmlTitleRe = regexp.MustCompile(`(?is)<title[^>]*>([^<]+)</title>`)
)

// resolveInviteTitleViaWeb 兜底解析: MTProto checkChatInvite 拿不到(如被踢/CHANNEL_PRIVATE)时,
// 抓 t.me 邀请页 og:title 补群名。纯 HTTP、无状态, 不会给协议号带来任何调用风险。
// 注意: 仅对"未隐藏预览"的群有效; 隐藏预览/封禁链接返回错误。
func resolveInviteTitleViaWeb(ctx context.Context, hash string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://t.me/+"+hash, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 t.me 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("t.me 返回 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return "", err
	}
	html := string(body)
	title := ""
	if m := ogTitleRe.FindStringSubmatch(html); len(m) > 1 {
		title = strings.TrimSpace(m[1])
	}
	// og:title 缺省或泛化时退回 <title>
	if title == "" || isGenericTitle(title) {
		if m := htmlTitleRe.FindStringSubmatch(html); len(m) > 1 {
			t := strings.TrimSpace(m[1])
			if !isGenericTitle(t) {
				title = t
			}
		}
	}
	if title == "" || isGenericTitle(title) {
		return "", fmt.Errorf("页面未暴露群名(群可能隐藏了预览)")
	}
	// t.me 标题常带 "Telegram: " 前缀, 去掉
	title = strings.TrimSpace(strings.TrimPrefix(title, "Telegram: "))
	return title, nil
}

// isGenericTitle 识别"未携带群名"的泛化标题
func isGenericTitle(t string) bool {
	s := strings.ToLower(strings.TrimSpace(t))
	if s == "" {
		return true
	}
	for _, k := range []string{"join group", "telegram", "sign in", "page not found", "webogram"} {
		if strings.Contains(s, k) && len(s) < 32 {
			return true
		}
	}
	return false
}
