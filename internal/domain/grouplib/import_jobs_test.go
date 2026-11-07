package grouplib

import (
	"errors"
	"testing"
	"time"

	"tg-sender/internal/domain"
)

func TestImportFailureLink(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		want   string
	}{
		{name: "public", detail: "[network] https://t.me/ExampleGroup: timeout", want: "https://t.me/ExampleGroup"},
		{name: "private invite", detail: "[expired] https://t.me/+AbCdEf123: invite expired", want: "https://t.me/+AbCdEf123"},
		{name: "database", detail: "[database] https://t.me/ExampleGroup: insert failed", want: "https://t.me/ExampleGroup"},
		{name: "malformed", detail: "[network] not-a-link: timeout", want: ""},
		{name: "missing classification", detail: "https://t.me/ExampleGroup: timeout", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := importFailureLink(tt.detail); got != tt.want {
				t.Fatalf("importFailureLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTelegramMTProtoRateLimit(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want time.Duration
	}{
		{name: "seconds", err: errors.New("rpc error: FLOOD_WAIT_37"), want: 37 * time.Second},
		{name: "minimum", err: errors.New("FLOOD_WAIT_0"), want: time.Second},
		{name: "not flood wait", err: errors.New("CHANNEL_PRIVATE")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := telegramMTProtoRateLimit(tt.err)
			if tt.want == 0 {
				if got != nil {
					t.Fatalf("telegramMTProtoRateLimit() = %v, want nil", got)
				}
				return
			}
			var rateLimitErr *TelegramRateLimitError
			if !errors.As(got, &rateLimitErr) {
				t.Fatalf("telegramMTProtoRateLimit() = %v, want TelegramRateLimitError", got)
			}
			if rateLimitErr.RetryAfter != tt.want {
				t.Fatalf("RetryAfter = %s, want %s", rateLimitErr.RetryAfter, tt.want)
			}
		})
	}
}

func TestSummarizeImportBatchRateLimitPreservesCompletedPrefix(t *testing.T) {
	result := &ImportResult{
		Processed: 3,
		Imported:  []domain.SystemGroup{{Title: "one"}},
		Failed:    []string{
			"[not_found] https://t.me/missinggroup: not found",
			"[rate_limited] https://t.me/currentgroup: FLOOD_WAIT_37",
		},
		Skipped: 4,
	}
	got := summarizeImportBatch(result, true)
	if got.processed != 3 || got.imported != 1 || got.failed != 1 || got.skipped != 0 {
		t.Fatalf("unexpected rate-limited progress: %+v", got)
	}
	if got.detail != "[not_found] https://t.me/missinggroup: not found" {
		t.Fatalf("unexpected completed-prefix detail: %q", got.detail)
	}
}

func TestSummarizeImportBatchNormalProgress(t *testing.T) {
	result := &ImportResult{
		Processed: 5,
		Imported: []domain.SystemGroup{{Title: "one"}, {Title: "two"}},
		Failed: []string{"[not_found] https://t.me/missinggroup: not found"},
		Skipped:   2,
	}
	got := summarizeImportBatch(result, false)
	if got.processed != 5 || got.imported != 2 || got.failed != 1 || got.skipped != 2 {
		t.Fatalf("unexpected progress: %+v", got)
	}
	if got.detail != result.Failed[0] {
		t.Fatalf("detail = %q, want %q", got.detail, result.Failed[0])
	}
}
