package middleware

import (
	"errors"
	"testing"
)

func TestBlacklistLookupRevokedFailsClosed(t *testing.T) {
	tests := []struct {
		name  string
		count int
		err   error
		want  bool
	}{
		{name: "listed token is revoked", count: 1, want: true},
		{name: "unlisted token is not revoked", count: 0, want: false},
		{name: "database lookup error is treated as revoked", err: errors.New("database unavailable"), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := blacklistLookupRevoked(tt.count, tt.err); got != tt.want {
				t.Fatalf("blacklistLookupRevoked(%d, %v) = %v, want %v", tt.count, tt.err, got, tt.want)
			}
		})
	}
}
