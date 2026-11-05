package grouplib

import (
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
