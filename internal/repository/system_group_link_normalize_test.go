package repository

import "testing"

func TestNormalizeSystemGroupLink(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "uppercase scheme and www host", input: "HTTPS://WWW.T.ME/ExampleGroup/", want: "t.me/examplegroup"},
		{name: "telegram alias and preview prefix", input: "http://www.telegram.me/s/ExampleGroup?single=1", want: "t.me/examplegroup"},
		{name: "joinchat invite alias", input: "https://t.me/joinchat/AbCdEf", want: "t.me/+AbCdEf"},
		{name: "invite hash remains case sensitive", input: "https://t.me/+AbCdEf", want: "t.me/+AbCdEf"},
		{name: "reject unrelated host", input: "https://example.com/t.me/group", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSystemGroupLink(tt.input); got != tt.want {
				t.Fatalf("normalizeSystemGroupLink(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
