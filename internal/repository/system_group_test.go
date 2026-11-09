package repository

import "testing"

func TestNormalizeSystemGroupLinkPreservesInviteHashCase(t *testing.T) {
	upper := normalizeSystemGroupLink("https://t.me/+AbCdEf")
	lower := normalizeSystemGroupLink("https://t.me/+abcdef")
	if upper == lower {
		t.Fatalf("private invite hashes are case-sensitive: %q == %q", upper, lower)
	}
	if upper != "t.me/+AbCdEf" {
		t.Fatalf("unexpected normalized invite: %q", upper)
	}
}

func TestNormalizeSystemGroupLinkCanonicalizesPublicUsernames(t *testing.T) {
	got := normalizeSystemGroupLink("https://Telegram.me/ExampleGroup/")
	if got != "t.me/examplegroup" {
		t.Fatalf("expected canonical public link, got %q", got)
	}
}

func TestNormalizeSystemGroupLinkRejectsForeignHosts(t *testing.T) {
	if got := normalizeSystemGroupLink("https://example.com/t.me/groupname"); got != "" {
		t.Fatalf("expected foreign host to be rejected, got %q", got)
	}
}
