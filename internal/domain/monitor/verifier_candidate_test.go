package monitor

import (
	"bytes"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestExtractCandidateReadsMTGoV025InlineButtonTypes(t *testing.T) {
	msg := &tg.Message{
		Message: "请完成验证",
		ReplyMarkup: &tg.ReplyInlineMarkup{
			Rows: []*tg.KeyboardInlineButtonRow{{
				Buttons: []*tg.KeyboardInlineButton{
					{
						Text: "我已加入",
						Type: &tg.InlineButtonTypeCallback{Data: []byte("confirm")},
					},
					{
						Text: "加入频道",
						Type: &tg.InlineButtonTypeURL{URL: "https://t.me/example"},
					},
					{
						Text: "打开验证",
						Type: &tg.InlineButtonTypeWebView{URL: "https://t.me/example?start=verify"},
					},
				},
			}},
		},
	}

	got := extractCandidate(msg)
	if len(got.buttons) != 3 {
		t.Fatalf("buttons = %d, want 3", len(got.buttons))
	}
	if got.buttons[0].text != "我已加入" || !bytes.Equal(got.buttons[0].data, []byte("confirm")) {
		t.Fatalf("callback button not extracted: %#v", got.buttons[0])
	}
	if got.buttons[1].text != "加入频道" || got.buttons[1].url != "https://t.me/example" {
		t.Fatalf("URL button not extracted: %#v", got.buttons[1])
	}
	if got.buttons[2].text != "打开验证" || got.buttons[2].url != "https://t.me/example?start=verify" {
		t.Fatalf("webview button not extracted: %#v", got.buttons[2])
	}
}
