package grouplib

import (
	"reflect"
	"testing"

	"tg-sender/internal/domain"
)

func TestFilterImportLinks(t *testing.T) {
	input := "https://t.me/ValidGroup\n@validgroup\nhttps://t.me/+AbCdEf123\nhttps://t.me/+abcdef123\nhttps://evil.com/t.me/fakegroup\nnot-a-link"
	got := filterImportLinks(input)
	want := []string{
		"https://t.me/ValidGroup",
		"https://t.me/+AbCdEf123",
		"https://t.me/+abcdef123",
	}
	if !reflect.DeepEqual(got.Links, want) {
		t.Fatalf("links = %#v, want %#v", got.Links, want)
	}
	if got.RawCount != 6 || got.Duplicate != 1 || got.Invalid != 2 {
		t.Fatalf("counts = raw:%d duplicate:%d invalid:%d, want raw:6 duplicate:1 invalid:2",
			got.RawCount, got.Duplicate, got.Invalid)
	}
}

func TestFilterImportLinksTelegramDeepLinksAndPreviewURLs(t *testing.T) {
	got := filterImportLinks("tg://resolve?domain=PublicGroup\ntg://join?invite=AbCdEf123\nhttps://t.me/s/ChannelName")
	want := []string{"https://t.me/PublicGroup", "https://t.me/+AbCdEf123", "https://t.me/ChannelName"}
	if !reflect.DeepEqual(got.Links, want) {
		t.Fatalf("links = %#v, want %#v", got.Links, want)
	}
}

func TestFilterImportLinksMixedCaseScheme(t *testing.T) {
	got := filterImportLinks("HTTPS://WWW.T.ME/ValidGroup")
	if len(got.Links) != 1 || got.Links[0] != "https://t.me/ValidGroup" {
		t.Fatalf("links = %#v, want normalized Telegram link", got.Links)
	}
}

func TestFilterImportLinksRejectsInvalidUsernameLength(t *testing.T) {
	got := filterImportLinks("@abcd\n@this_username_is_definitely_longer_than_32_chars")
	if len(got.Links) != 0 {
		t.Fatalf("expected no valid links, got %#v", got.Links)
	}
}

func TestFilterImportLinksExcludesBotUsernamesFromResumablePayload(t *testing.T) {
	got := filterImportLinks("https://t.me/real_group\nhttps://t.me/SomeBot\n@anotherbot\n@validchannel")
	want := []string{"https://t.me/real_group", "https://t.me/validchannel"}
	if !reflect.DeepEqual(got.Links, want) {
		t.Fatalf("links = %#v, want %#v", got.Links, want)
	}
	if got.RawCount != 4 || got.Invalid != 2 || got.Duplicate != 0 {
		t.Fatalf("counts = raw:%d duplicate:%d invalid:%d, want raw:4 duplicate:0 invalid:2", got.RawCount, got.Duplicate, got.Invalid)
	}
}

func TestSummarizeImportBatchRateLimitPersistsOnlyCompletedPrefix(t *testing.T) {
	result := &ImportResult{
		Processed: 2,
		Imported:  []domain.SystemGroup{{}, {}},
		Failed:    []string{"first link: unavailable", "third link: FLOOD_WAIT"},
		Skipped:   5, // includes parse-time skips from the unprocessed suffix
	}
	got := summarizeImportBatch(result, true)
	if got.processed != 2 || got.imported != 2 || got.failed != 1 || got.skipped != 0 {
		t.Fatalf("progress = %+v, want processed=2 imported=2 failed=1 skipped=0", got)
	}
	if got.detail != "first link: unavailable" {
		t.Fatalf("detail = %q, want only completed-prefix failure", got.detail)
	}
}

func TestSummarizeImportBatchKeepsSkippedForCompletedBatch(t *testing.T) {
	result := &ImportResult{
		Processed: 3,
		Imported:  []domain.SystemGroup{{}},
		Failed:    []string{"second link: unavailable", "third link: unavailable"},
		Skipped:   4,
	}
	got := summarizeImportBatch(result, false)
	if got.processed != 3 || got.imported != 1 || got.failed != 2 || got.skipped != 4 {
		t.Fatalf("progress = %+v, want processed=3 imported=1 failed=2 skipped=4", got)
	}
	if got.detail != "second link: unavailable\nthird link: unavailable" {
		t.Fatalf("detail = %q, want both completed failures", got.detail)
	}
}
