package grouplib

import (
	"errors"
	"testing"
)

func TestIsPermanentResolutionError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		inviteLink bool
		want       bool
	}{
		{name: "nil error", err: nil, inviteLink: true, want: false},
		{name: "invite expired", err: errors.New("邀请链接已过期"), inviteLink: true, want: true},
		{name: "invite invalid", err: errors.New("邀请链接无效"), inviteLink: true, want: true},
		{name: "public username not found", err: errors.New("页面返回 404"), inviteLink: false, want: true},
		{name: "gone page", err: errors.New("页面返回 410"), inviteLink: true, want: true},
		{name: "temporary server error", err: errors.New("页面返回 503"), inviteLink: false, want: false},
		{name: "network timeout", err: errors.New("请求超时"), inviteLink: true, want: false},
		{name: "expired wording on public link remains retryable", err: errors.New("内容已过期"), inviteLink: false, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPermanentResolutionError(tt.err, tt.inviteLink); got != tt.want {
				t.Fatalf("isPermanentResolutionError(%v, %v) = %v, want %v", tt.err, tt.inviteLink, got, tt.want)
			}
		})
	}
}
