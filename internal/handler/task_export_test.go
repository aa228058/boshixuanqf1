package handler

import "testing"

func TestCSVFieldNeutralizesSpreadsheetFormulas(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "equals", input: "=1+1", want: "\t=1+1"},
		{name: "plus", input: "+1", want: "\t+1"},
		{name: "minus", input: "-1", want: "\t-1"},
		{name: "at", input: "@SUM(A1)", want: "\t@SUM(A1)"},
		{name: "leading spaces", input: "  =1+1", want: "\t  =1+1"},
		{name: "leading control", input: "\r\n=1+1", want: "\"\t\r\n=1+1\""},
		{name: "unicode space", input: "\u00a0=1+1", want: "\t\u00a0=1+1"},
		{name: "unicode format control", input: "\u200b=1+1", want: "\t\u200b=1+1"},
		{name: "safe url", input: "https://example.com/a,b", want: "\"https://example.com/a,b\""},
		{name: "quoted field", input: "a,\"b\"", want: "\"a,\"\"b\"\"\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := csvField(tt.input); got != tt.want {
				t.Fatalf("csvField(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
