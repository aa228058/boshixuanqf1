package task

import (
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tgerr"
)

func TestIsDefiniteInviteRPCError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "Telegram client rejection", err: tgerr.New(400, "USER_PRIVACY_RESTRICTED"), want: true},
		{name: "Telegram flood wait", err: tgerr.New(420, "FLOOD_WAIT_60"), want: true},
		{name: "Telegram server error is ambiguous", err: tgerr.New(500, "INTERNAL"), want: false},
		{name: "transport error is ambiguous", err: errors.New("connection reset by peer"), want: false},
		{name: "nil error", err: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDefiniteInviteRPCError(tt.err); got != tt.want {
				t.Fatalf("isDefiniteInviteRPCError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
