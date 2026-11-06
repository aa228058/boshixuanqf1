package task

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestDefaultFreeSpeechStatus(t *testing.T) {
	cases := []struct {
		name   string
		rights *tg.ChatBannedRights
		want   string
	}{
		{name: "no default restrictions", rights: nil, want: "allowed"},
		{name: "text messages allowed", rights: &tg.ChatBannedRights{SendMessages: false}, want: "allowed"},
		{name: "default send restriction", rights: &tg.ChatBannedRights{SendMessages: true}, want: "restricted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := defaultFreeSpeechStatus(tc.rights); got != tc.want {
				t.Fatalf("defaultFreeSpeechStatus()=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsJoinCheckpointStatus(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{status: "success", want: true},
		{status: "pending_approval", want: true},
		{status: "failed", want: false},
		{status: "info", want: false},
		{status: "", want: false},
	}
	for _, tc := range cases {
		if got := isJoinCheckpointStatus(tc.status); got != tc.want {
			t.Errorf("isJoinCheckpointStatus(%q)=%v, want %v", tc.status, got, tc.want)
		}
	}
}
