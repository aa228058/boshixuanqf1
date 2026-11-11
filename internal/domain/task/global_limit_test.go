package task

import (
	"errors"
	"testing"
)

func TestShouldDeferGlobalClaim(t *testing.T) {
	tests := []struct {
		name    string
		running int
		limit   int
		err     error
		want    bool
	}{
		{name: "at limit", running: 5, limit: 5, want: true},
		{name: "above limit", running: 6, limit: 5, want: true},
		{name: "below limit", running: 4, limit: 5, want: false},
		{name: "count failure fails closed", limit: 5, err: errors.New("database unavailable"), want: true},
		{name: "disabled limit", running: 100, limit: 0, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldDeferGlobalClaim(tt.running, tt.limit, tt.err); got != tt.want {
				t.Fatalf("shouldDeferGlobalClaim(%d, %d, %v) = %v, want %v", tt.running, tt.limit, tt.err, got, tt.want)
			}
		})
	}
}
