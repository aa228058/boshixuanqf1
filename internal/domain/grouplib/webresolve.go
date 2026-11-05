package grouplib

import (
	"context"
	"fmt"
	htmlstd "html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	metaTagRe   = regexp.MustCompile(\`(?is)<meta\\b[^>]*>\`)
	attrRe      = regexp.MustCompile(\`(?is)([a-z_:][-a-z0-9_:]*)\\s*=\\s*(?:"([^"]*)"|'([^']*)'|([^\\s>]+))\`)
	htmlTitleRe = regexp.MustCompile(\`(?is)<title[^>]*>(.*?)</title>\`)
	pageExtraRe = regexp.MustCompile(\`(?is)<[^>]+class=["'][^"']*tgme_page_extra[^"']*["'][^>]*>(.*?)</[^>]+>\`)
	htmlTagRe   = regexp.MustCompile(\`<[^>]+>\`)
	digitsRe    = regexp.MustCompile(\`[0-9][0-9\\s,.\\x{00a0}\\x{202f}]*\`)
)

const telegramMobileUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1"

func resolveInviteTitleViaWeb(ctx context.Context, hash string) (string, error) {
	title, _, err := resolveInviteInfoViaWeb(ctx, hash)
	return title, err
}

func resolveInviteInfoViaWeb(ctx context.Context, hash string) (string, int, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" || strings.ContainsAny(hash, "/?#") {
		return "", 0, fmt.Errorf("邀请 hash 格式无效")
	}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	target := "https://t.me/+" + url.PathEscape(hash)
	client := &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("Telegram 页面重定向次数过多")
			}
			host := strings.ToLower(req.URL.Hostname())
			if host != "t.me" && host != "www.t.me" {
				return fmt.Errorf("拒绝跳转到非 Telegram 域名")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", telegramMobileUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("请求 t.me 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("t.me 返回 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024+1))
	if err != nil {
		return "", 0, err
	}
	if len(body) > 512*1024 {
		return "", 0, fmt.Errorf("Telegram 网页内容超过限制")
	}
	page := string(body)

	title := extractTelegramMeta(page, "og:title")
	if title == "" || isGenericTitle(title) {
		if match := htmlTitleRe.FindStringSubmatch(page); len(match) > 1 {
			candidate := cleanTelegramText(match[1])
			if !isGenericTitle(candidate) {
				title = candidate
			}
		}
	}
	title = strings.TrimSpace(strings.TrimPrefix(title, "Telegram:"))
	if isGenericTitle(title) {
		title = ""
	}

	memberCount := 0
	if match := pageExtraRe.FindStringSubmatch(page); len(match) > 1 {
		memberCount = parseWebMemberCount(cleanTelegramText(match[1]))
	}
	if title == "" && memberCount == 0 {
		return "", 0, fmt.Errorf("页面未暴露群组资料，可能隐藏预览或需要客户端验证")
	}
	return title, memberCount, nil
}

func extractTelegramMeta(page, property string) string {
	for _, tag := range metaTagRe.FindAllString(page, -1) {
		attrs := map[string]string{}
		for _, match := range attrRe.FindAllStringSubmatch(tag, -1) {
			value := match[2]
			if value == "" {
				value = match[3]
			}
			if value == "" {
				value = match[4]
			}
			attrs[strings.ToLower(match[1])] = htmlstd.UnescapeString(value)
		}
		if strings.EqualFold(attrs["property"], property) || strings.EqualFold(attrs["name"], property) {
			return cleanTelegramText(attrs["content"])
		}
	}
	return ""
}

func cleanTelegramText(value string) string {
	value = htmlstd.UnescapeString(htmlTagRe.ReplaceAllString(value, " "))
	return strings.Join(strings.Fields(value), " ")
}

func parseWebMemberCount(value string) int {
	lower := strings.ToLower(value)
	labels := []string{" member", " members", " subscriber", " subscribers", " участник", " участника", " участников", " membre", " membres"}
	found := false
	for _, label := range labels {
		if strings.Contains(lower, label) {
			found = true
			break
		}
	}
	if !found {
		return 0
	}
	raw := digitsRe.FindString(value)
	if raw == "" {
		return 0
	}
	raw = strings.NewReplacer(" ", "", ",", "", ".", "", " ", "", " ", "").Replace(raw)
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func isGenericTitle(value string) bool {
	s := strings.ToLower(strings.TrimSpace(value))
	if s == "" {
		return true
	}
	for _, word := range []string{"join group", "telegram", "sign in", "page not found", "webogram", "加入群组", "登录"} {
		if strings.Contains(s, word) && len(s) < 48 {
			return true
		}
	}
	return false
}

func resolvePublicGroupInfoViaWeb(ctx context.Context, username string) (string, int, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if username == "" || strings.ContainsAny(username, "/?#") {
		return "", 0, fmt.Errorf("公开群 username 格式无效")
	}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	target := "https://t.me/" + url.PathEscape(username)
	client := &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("Telegram 页面重定向次数过多")
			}
			host := strings.ToLower(req.URL.Hostname())
			if host != "t.me" && host != "www.t.me" {
				return fmt.Errorf("拒绝跳转到非 Telegram 域名")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", telegramMobileUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("请求 Telegram 公开群页面失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("Telegram 网页返回 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024+1))
	if err != nil {
		return "", 0, err
	}
	if len(body) > 512*1024 {
		return "", 0, fmt.Errorf("Telegram 网页内容超过限制")
	}
	page := string(body)
	title := extractTelegramMeta(page, "og:title")
	if title == "" || isGenericTitle(title) {
		if match := htmlTitleRe.FindStringSubmatch(page); len(match) > 1 {
			candidate := cleanTelegramText(match[1])
			if !isGenericTitle(candidate) {
				title = candidate
			}
		}
	}
	title = strings.TrimSpace(strings.TrimPrefix(title, "Telegram:"))
	if isGenericTitle(title) {
		title = ""
	}
	count := 0
	if match := pageExtraRe.FindStringSubmatch(page); len(match) > 1 {
		count = parseWebMemberCount(cleanTelegramText(match[1]))
	}
	if title == "" && count == 0 {
		return "", 0, fmt.Errorf("页面未暴露公开群资料")
	}
	return title, count, nil
}
