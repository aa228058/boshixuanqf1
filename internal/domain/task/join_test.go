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
