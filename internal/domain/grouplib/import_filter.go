package grouplib

import (
	"net/url"
	"regexp"
	"strings"
)

// importCandidateRe 只识别 Telegram 公共用户名和私密邀请链接。
// 公共用户名遵循 Telegram 规则(5-32位, 字母开头); 邀请 hash 单独校验。
var importCandidateRe = regexp.MustCompile(`(?i)(?:(?:https?://)?(?:www\.)?(?:t\.me|telegram\.me|telegram\.dog)/(?:s/|joinchat/|\+)?[A-Za-z0-9_-]{5,64}|tg://(?:resolve\?domain=[A-Za-z][A-Za-z0-9_]{4,31}|join\?invite=[A-Za-z0-9_-]{5,64})|@[A-Za-z][A-Za-z0-9_]{4,31})`)

type importFilterResult struct {
	Links     []string
	RawCount  int
	Duplicate int
	Invalid   int
}

// filterImportLinks 是纯格式筛选，不访问 Telegram、不创建群库记录。
// 只有确认后的 Links 才会进入后台解析队列。
func filterImportLinks(payload string) importFilterResult {
	tokens := splitImportLines(payload)
	result := importFilterResult{RawCount: len(tokens)}
	seen := make(map[string]struct{}, len(tokens))
	candidates := 0

	for _, token := range tokens {
		token = strings.Trim(token, `"'()[]{}<>，。！？、`)
		loc := importCandidateRe.FindStringIndex(token)
		if loc == nil {
			continue
		}
		// 防止从 other.example/t.me/name 这类外站路径中截出伪 Telegram 链接。
		if loc[0] > 0 {
			prev := token[loc[0]-1]
			if (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z') ||
				(prev >= '0' && prev <= '9') || strings.ContainsRune("./_-", rune(prev)) {
				continue
			}
		}
		// 正则可能匹配到超长用户名的合法前缀；若后面仍是用户名字符，必须整段拒绝。
		if loc[1] < len(token) {
			next := token[loc[1]]
			if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') ||
				(next >= '0' && next <= '9') || next == '_' || next == '-' {
				continue
			}
		}
		candidate := token[loc[0]:loc[1]]
		normalized, key, ok := normalizeImportCandidate(candidate)
		if !ok {
			continue
		}
		// Bot usernames are syntactically valid Telegram handles but are not
		// group/channel targets. Filter them before creating the resumable job
		// payload so worker offsets always refer to one import candidate per line.
		if strings.HasPrefix(key, "u:") && strings.HasSuffix(strings.ToLower(strings.TrimPrefix(key, "u:")), "bot") {
			continue
		}
		candidates++
		if _, exists := seen[key]; exists {
			result.Duplicate++
			continue
		}
		seen[key] = struct{}{}
		result.Links = append(result.Links, normalized)
	}
	result.Invalid = result.RawCount - candidates
	if result.Invalid < 0 {
		result.Invalid = 0
	}
	return result
}

func normalizeImportCandidate(candidate string) (normalized, key string, ok bool) {
	if strings.HasPrefix(strings.ToLower(candidate), "tg://") {
		u, err := url.Parse(candidate)
		if err != nil {
			return "", "", false
		}
		if strings.EqualFold(u.Host, "resolve") {
			username := u.Query().Get("domain")
			if len(username) < 5 || len(username) > 32 {
				return "", "", false
			}
			return "https://t.me/" + username, "u:" + strings.ToLower(username), true
		}
		if strings.EqualFold(u.Host, "join") {
			hash := u.Query().Get("invite")
			if len(hash) < 5 || len(hash) > 64 {
				return "", "", false
			}
			return "https://t.me/+" + hash, "p:" + hash, true
		}
		return "", "", false
	}
	if strings.HasPrefix(candidate, "@") {
		username := candidate[1:]
		if len(username) < 5 || len(username) > 32 {
			return "", "", false
		}
		return "https://t.me/" + username, "u:" + strings.ToLower(username), true
	}

	value := candidate
	lowerValue := strings.ToLower(value)
	if strings.HasPrefix(lowerValue, "https://") {
		value = value[len("https://"):]
	} else if strings.HasPrefix(lowerValue, "http://") {
		value = value[len("http://"):]
	}
	if strings.HasPrefix(strings.ToLower(value), "www.") {
		value = value[len("www."):]
	}
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	host := strings.ToLower(parts[0])
	if host != "t.me" && host != "telegram.me" && host != "telegram.dog" {
		return "", "", false
	}
	path := parts[1]
	if strings.HasPrefix(strings.ToLower(path), "s/") {
		path = path[2:]
	}
	lowerPath := strings.ToLower(path)
	if strings.HasPrefix(path, "+") || strings.HasPrefix(lowerPath, "joinchat/") {
		hash := path
		if strings.HasPrefix(lowerPath, "joinchat/") {
			hash = path[len("joinchat/"):]
		} else {
			hash = strings.TrimPrefix(hash, "+")
		}
		if len(hash) < 5 || len(hash) > 64 {
			return "", "", false
		}
		// 私密邀请 hash 区分大小写。
		return "https://t.me/+" + hash, "p:" + hash, true
	}
	username := path
	if len(username) < 5 || len(username) > 32 || username[0] < 'A' || (username[0] > 'Z' && username[0] < 'a') || username[0] > 'z' {
		return "", "", false
	}
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return "", "", false
		}
	}
	return "https://t.me/" + username, "u:" + strings.ToLower(username), true
}
