package grouplib

import (
	"fmt"
	"strings"
	"testing"
)

// benchmarkImportPayload builds a deterministic mixed corpus. It deliberately
// measures only local parsing/deduplication: no Telegram or database calls.
func benchmarkImportPayload(n int) string {
	var payload strings.Builder
	payload.Grow(n * 40)
	for i := 0; i < n; i++ {
		switch i % 4 {
		case 0, 1:
			fmt.Fprintf(&payload, "https://t.me/BenchGroup%08d\n", i)
		case 2:
			// Duplicate the immediately preceding valid public link.
			fmt.Fprintf(&payload, "https://t.me/BenchGroup%08d\n", i-1)
		case 3:
			fmt.Fprintf(&payload, "not-a-telegram-link-%08d\n", i)
		}
	}
	return payload.String()
}

func BenchmarkFilterImportLinks(b *testing.B) {
	for _, size := range []int{10_000, 30_000, 50_000} {
		b.Run(fmt.Sprintf("links_%d", size), func(b *testing.B) {
			payload := benchmarkImportPayload(size)
			b.SetBytes(int64(len(payload)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := filterImportLinks(payload)
				if len(result.Links) != size/2 || result.Duplicate != size/4 || result.Invalid != size/4 {
					b.Fatalf("unexpected corpus counts: valid=%d duplicate=%d invalid=%d",
						len(result.Links), result.Duplicate, result.Invalid)
				}
			}
		})
	}
}
