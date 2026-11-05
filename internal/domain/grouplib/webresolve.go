package grouplib

import (
	"context"
	"fmt"
	"html"
	"io"
	"strconv"
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
	pageExtraRe = regexp.MustCompile(`(?is)<[^>]+class=["\'][^"\']*tgme_page_extra[^"\']*["\'][^>]*>(.*?)</[^>]+>`)
	digitsRe = regexp.MustCompile(`[0-9][0-9\\s,.\\x{00a0}\\x{202f}]*`)
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1")
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
	memberCount := 0
	if extra := pageExtraRe.FindStringSubmatch(html); len(extra) > 1 {
		memberCount = parseWebMemberCount(html.UnescapeString(regexp.MustCompile(`<[^>]+>`).ReplaceAllString(extra[1], " ")))
	}
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
	if title == "" && memberCount == 0 { return "", fmt.Errorf("页面未暴露群组资料(可能隐藏预览)") }
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


// parseWebMemberCount 只识别成员/订阅者标签，不把在线人数当成员数。
func parseWebMemberCount(value string) int {
	lower := strings.ToLower(value)
	labels := []string{" member", " members", " subscriber", " subscribers", " участник", " участника", " участников", " membre", " membres"}
	found := false
	for _, label := range labels {
		if strings.Contains(lower, label) { found = true; break }
	}
	if !found { return 0 }
	raw := digitsRe.FindString(value)
	if raw == "" { return 0 }
	raw = strings.NewReplacer(" ", "", ",", "", ".", "", "\u00a0", "", "\u202f", "").Replace(raw)
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 { return 0 }
	return n
}
