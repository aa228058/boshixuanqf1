package grouplib

import (
	"net/http"
	"testing"
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
