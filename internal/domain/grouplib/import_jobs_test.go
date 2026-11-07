package grouplib

import "testing"

func TestImportFailureLink(t *testing.T) {
	tests := []struct {
		name string
		detail string
		want string
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
