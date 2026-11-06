package grouplib

import (
	"errors"
	"net/http"
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
	if req.Header.Get("User-Agent") != telegramMobileUA {
		t.Fatal("expected the configured stable User-Agent")
	}
	if req.Header.Get("Accept-Language") == "" || req.Header.Get("Accept") == "" {
		t.Fatal("expected Accept and Accept-Language headers")
	}
	if req.Header.Get("Referer") != "" {
		t.Fatal("Referer should not be fabricated")
	}
}

func TestRandomResolveDelayBounds(t *testing.T) {
	for i := 0; i < 1000; i++ {
		delay := randomResolveDelay()
		if delay < 5*time.Second || delay > 15*time.Second {
			t.Fatalf("randomResolveDelay()=%s, want range [5s,15s]", delay)
		}
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
