package task

import "testing"

func TestExtractBioTelegramLinks(t *testing.T) {
	tests := []struct {
		name string
		bio  string
		want []bioLinkCandidate
	}{
		{
			name: "public and private links",
			bio:  "频道 https://t.me/example_group and invite https://t.me/+AbCdEf123",
			want: []bioLinkCandidate{
				{LinkType: "public", LinkHash: "example_group", FullURL: "https://t.me/example_group"},
				{LinkType: "private", LinkHash: "AbCdEf123", FullURL: "https://t.me/+AbCdEf123"},
			},
		},
		{
			name: "joinchat and duplicate",
			bio:  "t.me/joinchat/AbCdEf123, https://telegram.me/joinchat/AbCdEf123",
			want: []bioLinkCandidate{
				{LinkType: "private", LinkHash: "AbCdEf123", FullURL: "https://t.me/joinchat/AbCdEf123"},
			},
		},
		{
			name: "ignore utility and message paths",
			bio:  "t.me/addstickers/foo t.me/share/url?url=x t.me/c/123/456",
		},
		{
			name: "ignore malformed public usernames",
			bio:  "t.me/abc t.me/a-very-long-name-that-is-definitely-over-thirty-two-characters",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBioTelegramLinks(tt.bio)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d candidates (%#v), want %d (%#v)", len(got), got, len(tt.want), tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("candidate %d = %#v, want %#v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
