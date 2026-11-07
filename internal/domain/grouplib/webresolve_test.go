package grouplib

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestExtractTelegramMetaAttributeOrder(t *testing.T) {
	page := `<html><head><meta content="A &amp; B" property="og:title"></head></html>`
	if got := extractTelegramMeta(page, "og:title"); got != "A & B" {
		t.Fatalf("unexpected title: %q", got)
	}
}

func TestParseWebMemberCount(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"12 345 members", 12345},
		{"1,234 subscribers", 1234},
		{"1.2K members", 1200},
		{"3,4M subscribers", 3400000},
		{"850 members", 850},
		{"1.2K 成员", 1200},
		{"2 001 участников", 2001},
		{"42 online", 0},
		{"no preview", 0},
	}
	for _, tc := range cases {
		if got := parseWebMemberCount(tc.input); got != tc.want {
			t.Errorf("parseWebMemberCount(%q)=%d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestIsInvalidInvitePage(t *testing.T) {
	if !isInvalidInvitePage("<div>This invite link is invalid or has expired</div>") {
		t.Fatal("expected expired invite page to be rejected")
	}
	if isInvalidInvitePage("<a>Join Group</a><span>120 members</span>") {
		t.Fatal("valid invite preview must not be rejected")
	}
}

func TestDetectTelegramPageTypeRequiresExplicitActionLabel(t *testing.T) {
	cases := []struct {
		page string
		want string
	}{
		{page: "<a>Join Group</a>", want: "group"},
		{page: "<a>View in Telegram</a>", want: "group"},
		{page: "<a>Join Channel</a>", want: "channel"},
		{page: "<a>Preview channel</a>", want: "channel"},
		{page: "<a>Send Message</a>", want: "user"},
		{page: "<a>Start Bot</a>", want: "user"},
		{page: "<title>Telegram: Some Place</title>", want: "unknown"},
	}
	for _, tc := range cases {
		if got := detectTelegramPageType(tc.page); got != tc.want {
			t.Errorf("detectTelegramPageType(%q)=%q, want %q", tc.page, got, tc.want)
		}
	}
}

func TestInviteHashFromLink(t *testing.T) {
	cases := map[string]string{
		"https://t.me/+AbCdEf?single": "AbCdEf",
		"t.me/joinchat/AbCdEf":        "AbCdEf",
		"https://telegram.me/+XyZ":    "XyZ",
	}
	for input, want := range cases {
		if got := inviteHashFromLink(input); got != want {
			t.Errorf("inviteHashFromLink(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestApplyTelegramWebHeaders(t *testing.T) {
	req, err := http.NewRequest("GET", "https://t.me/example", nil)
	if err != nil {
		t.Fatal(err)
	}
	applyTelegramWebHeaders(req)
	if req.Header.Get("User-Agent") != telegramWebUserAgent {
		t.Fatal("expected the configured stable User-Agent")
	}
	if req.Header.Get("Accept-Language") == "" || req.Header.Get("Accept") == "" {
		t.Fatal("expected Accept and Accept-Language headers")
	}
	if strings.Contains(req.Header.Get("User-Agent"), "iPhone") || strings.Contains(req.Header.Get("User-Agent"), "Safari") {
		t.Fatal("User-Agent must identify this application, not impersonate a device or browser")
	}
	if req.Header.Get("Referer") != "" {
		t.Fatal("Referer should not be fabricated")
	}
}

func TestParseTelegramRetryAfter(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	date := now.Add(90 * time.Second).Format(http.TimeFormat)
	cases := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "seconds", value: "45", want: 45 * time.Second},
		{name: "http date", value: date, want: 90 * time.Second},
		{name: "past date", value: now.Add(-time.Minute).Format(http.TimeFormat), want: time.Second},
		{name: "missing", value: "", want: defaultTelegramRetryAfter},
		{name: "invalid", value: "later", want: defaultTelegramRetryAfter},
		{name: "negative", value: "-5", want: defaultTelegramRetryAfter},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseTelegramRetryAfter(tc.value, now); got != tc.want {
				t.Fatalf("parseTelegramRetryAfter(%q)=%s, want %s", tc.value, got, tc.want)
			}
		})
	}
}

func TestTelegramHTTPStatusErrorRateLimit(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{"30"}},
	}
	err := telegramHTTPStatusError(resp, "test")
	var rateLimitErr *TelegramRateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected TelegramRateLimitError, got %T: %v", err, err)
	}
	if rateLimitErr.RetryAfter != 30*time.Second {
		t.Fatalf("RetryAfter=%s, want 30s", rateLimitErr.RetryAfter)
	}
	if rateLimitErr.Endpoint != "test" {
		t.Fatalf("Endpoint=%q, want test", rateLimitErr.Endpoint)
	}

}

func TestTelegramResolverRejectsMalformedTargets(t *testing.T) {
	for _, username := range []string{"", "abcd", "bad/name", "bad?name", "bad.name", "bad-name"} {
		if telegramUsernameRe.MatchString(username) {
			t.Errorf("username %q should be rejected", username)
		}
	}
	for _, hash := range []string{"", "abcd", "bad/hash", "bad?hash", "bad.hash", strings.Repeat("a", 65)} {
		if telegramInviteHashRe.MatchString(hash) {
			t.Errorf("invite hash %q should be rejected", hash)
		}
	}
	if !telegramUsernameRe.MatchString("Valid_Group123") {
		t.Fatal("valid Telegram username was rejected")
	}
	if !telegramInviteHashRe.MatchString("AbCd_123-xy") {
		t.Fatal("valid invite hash was rejected")
	}
}

func TestTelegramRedirectRequiresSecureTelegramOrigin(t *testing.T) {
	for _, target := range []string{
		"http://t.me/groupname",
		"https://evil.example/groupname",
		"https://t.me:8443/groupname",
		"https://user@t.me/groupname",
	} {
		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := telegramWebClient.CheckRedirect(req, nil); err == nil {
			t.Errorf("redirect to %q should be rejected", target)
		}
	}
	for _, target := range []string{"https://t.me/groupname", "https://www.t.me/groupname", "https://t.me:443/groupname"} {
		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := telegramWebClient.CheckRedirect(req, nil); err != nil {
			t.Errorf("redirect to %q should be allowed: %v", target, err)
		}
	}
}
