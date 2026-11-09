package grouplib

import "testing"

func TestIsEmailAddressMention(t *testing.T) {
	tests := []struct {
		name string
		bio  string
		at   int
		end  int
		want bool
	}{
		{name: "email with uncommon TLD", bio: "contact user@example.dev", at: 12, end: 20, want: true},
		{name: "email with plus local part", bio: "reach me at name+tag@example.ai", at: 20, end: 28, want: true},
		{name: "email with dotted local part", bio: "contact first.last@example.dev", at: 19, end: 27, want: true},
		{name: "email with hyphenated local part", bio: "contact first-last@example.dev", at: 19, end: 27, want: true},
		{name: "email with underscore local part", bio: "contact first_last@example.dev", at: 19, end: 27, want: true},
		{name: "telegram handle after sentence punctuation", bio: "contact: @sample_group", at: 10, end: 23, want: false},
		{name: "standalone email-like domain", bio: "@example.com", at: 0, end: 8, want: true},
		{name: "telegram handle after whitespace", bio: "join @sample_group today", at: 5, end: 18, want: false},
		{name: "telegram handle in parentheses", bio: "(@sample_group)", at: 1, end: 14, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEmailAddressMention(tt.bio, tt.at, tt.end); got != tt.want {
				t.Fatalf("isEmailAddressMention(%q, %d, %d) = %v, want %v", tt.bio, tt.at, tt.end, got, tt.want)
			}
		})
	}
}
