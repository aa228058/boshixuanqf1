package domain

import "testing"

func TestSystemGroupDedupKeyPreservesInviteHashCase(t *testing.T) {
	upper := SystemGroup{TmeLink: "https://t.me/+AbCdEf"}
	lower := SystemGroup{TmeLink: "https://t.me/+abcdef"}
	if upper.DedupKey() == lower.DedupKey() {
		t.Fatal("case-sensitive invite hashes must not collapse into one provisional group")
	}
}

func TestSystemGroupDedupKeyNormalizesPublicUsername(t *testing.T) {
	upper := SystemGroup{Username: "@ExampleGroup"}
	lower := SystemGroup{Username: "examplegroup"}
	if upper.DedupKey() != lower.DedupKey() {
		t.Fatal("public usernames should be case-insensitive")
	}
}
