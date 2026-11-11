package grouplib

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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

func TestFilterImportLinksRejectsPartialUsernameMatches(t *testing.T) {
	input := "https://t.me/ValidGroup.evil\nhttps://t.me/ValidGroup%2Fother\nhttps://t.me/ValidGroup\nhttps://t.me/s/ChannelName"
	got := filterImportLinks(input)
	want := []string{"https://t.me/ValidGroup", "https://t.me/ChannelName"}
	if !reflect.DeepEqual(got.Links, want) {
		t.Fatalf("links = %#v, want %#v", got.Links, want)
	}
	if got.RawCount != 4 || got.Invalid != 2 || got.Duplicate != 0 {
		t.Fatalf("counts = raw:%d duplicate:%d invalid:%d, want raw:4 duplicate:0 invalid:2",
			got.RawCount, got.Duplicate, got.Invalid)
	}
}

func TestImportLinkRegexAndValidationSupportPreviewURLs(t *testing.T) {
	matches := importLinkRe.FindAllStringSubmatch("https://t.me/s/ChannelName https://t.me/+AbCdEf123 @ValidGroup", -1)
	if len(matches) != 3 {
		t.Fatalf("matches = %#v, want 3 Telegram targets", matches)
	}
	if matches[0][1] != "s/" || !telegramUsernameRe.MatchString(matches[0][2]) {
		t.Fatalf("preview URL was not recognized as a public username: %#v", matches[0])
	}
	if matches[1][1] != "+" || !telegramInviteHashRe.MatchString(matches[1][2]) {
		t.Fatalf("invite URL was not recognized: %#v", matches[1])
	}
	if !telegramUsernameRe.MatchString(matches[2][2]) {
		t.Fatalf("username was not validated: %#v", matches[2])
	}
}

func TestNormalizeImportCandidateRejectsMalformedTelegramDeepLinks(t *testing.T) {
	invalid := []string{
		"tg://resolve?domain=validgroup&phone=123",
		"tg://resolve?domain=validgroup%2Fevil",
		"tg://resolve?domain=validgroup#fragment",
		"tg://join?invite=AbCdEf123&extra=1",
		"tg://join?invite=bad%2Fhash",
		"tg://join/path?invite=AbCdEf123",
	}
	for _, candidate := range invalid {
		if normalized, _, ok := normalizeImportCandidate(candidate); ok {
			t.Errorf("normalizeImportCandidate(%q) = %q, want rejection", candidate, normalized)
		}
	}
}

func TestFilterImportLinksRejectsMalformedDeepLinkParameters(t *testing.T) {
	got := filterImportLinks("tg://resolve?domain=validgroup&phone=123\ntg://join?invite=AbCdEf123&extra=1\ntg://resolve?domain=ValidGroup")
	want := []string{"https://t.me/ValidGroup"}
	if !reflect.DeepEqual(got.Links, want) {
		t.Fatalf("links = %#v, want %#v", got.Links, want)
	}
}

func TestFilterImportLinksRejectsTruncatedPathAndQuery(t *testing.T) {
	got := filterImportLinks("https://t.me/ValidGroup/extra\nhttps://t.me/ValidGroup?start=abc\nhttps://t.me/ValidGroup#fragment\nhttps://t.me/ValidGroup")
	want := []string{"https://t.me/ValidGroup"}
	if !reflect.DeepEqual(got.Links, want) {
		t.Fatalf("links = %#v, want %#v", got.Links, want)
	}
	if got.Invalid != 3 {
		t.Fatalf("invalid = %d, want 3", got.Invalid)
	}
}

func TestResolveCachedImportTargetDeduplicatesConcurrentRequests(t *testing.T) {
	engine := &Engine{}
	var calls int32
	started := make(chan struct{})
	release := make(chan struct{})
	resolve := func() (string, int, string, int64, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			close(started)
		}
		<-release
		return "Cached Group", 42, "supergroup", 123, nil
	}
	var wg sync.WaitGroup
	results := make([]string, 2)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			title, _, _, _, err := engine.resolveCachedImportTarget(context.Background(), "u:cachedgroup", resolve)
			if err != nil {
				t.Errorf("resolveCachedImportTarget: %v", err)
				return
			}
			results[i] = title
		}(i)
	}
	<-started
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("resolver calls=%d, want 1", got)
	}
	for _, title := range results {
		if title != "Cached Group" {
			t.Fatalf("cached title=%q, want Cached Group", title)
		}
	}
}

func TestClassifyImportFailure(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"expired", errors.New("邀请已过期"), "expired"},
		{"invalid", errors.New("邀请无效"), "invalid"},
		{"rate limit", &TelegramRateLimitError{RetryAfter: time.Minute}, "rate_limited"},
		{"not found", &TelegramHTTPStatusError{StatusCode: 404}, "not_found"},
		{"forbidden", &TelegramHTTPStatusError{StatusCode: 403}, "forbidden"},
		{"transient", &TelegramHTTPStatusError{StatusCode: 502}, "transient"},
		{"unknown", errors.New("unexpected resolver response"), "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyImportFailure(tc.err); got != tc.want {
				t.Fatalf("classifyImportFailure=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsTransientImportDatabaseError(t *testing.T) {
	if !isTransientImportDatabaseError(&pgconn.PgError{Code: "40001"}) {
		t.Fatal("serialization failures should be retried")
	}
	if !isTransientImportDatabaseError(&pgconn.PgError{Code: "08006"}) {
		t.Fatal("connection failures should be retried")
	}
	if !isTransientImportDatabaseError(&pgconn.PgError{Code: "57P01"}) {
		t.Fatal("database shutdown should be retried")
	}
	if isTransientImportDatabaseError(&pgconn.PgError{Code: "23505"}) {
		t.Fatal("unique constraint violations should not be retried")
	}
}
