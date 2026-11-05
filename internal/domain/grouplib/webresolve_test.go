package grouplib

import "testing"

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
