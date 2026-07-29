package asc

import "testing"

func TestSanitizeTerminalTextRemovesInterpretedSequences(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text is unchanged",
			input: "screenshot-6.5-inch.png",
			want:  "screenshot-6.5-inch.png",
		},
		{
			name:  "escape introducer",
			input: "before\x1b[31mafter",
			want:  "before[31mafter",
		},
		{
			name:  "osc window title with bell terminator",
			input: "shot\x1b]0;pwned\x07.png",
			want:  "shot]0;pwned.png",
		},
		{
			name:  "c1 control sequence introducer",
			input: "state\u009b2Kdone",
			want:  "state2Kdone",
		},
		{
			name:  "c1 operating system command",
			input: "state\u009d0;pwned\u009cdone",
			want:  "state0;pwneddone",
		},
		{
			name:  "carriage return and del",
			input: "visible\rhidden\u007f",
			want:  "visiblehidden",
		},
		{
			name:  "bidi override",
			input: "gpj.\u202egnp.exe",
			want:  "gpj.gnp.exe",
		},
		{
			name:  "bidi isolates",
			input: "a\u2066b\u2067c\u2068d\u2069e",
			want:  "abcde",
		},
		{
			name:  "bidi marks",
			input: "a\u200eb\u200fc",
			want:  "abc",
		},
		{
			name:  "arabic letter mark",
			input: "total\u061c123",
			want:  "total123",
		},
		{
			name:  "unicode line and paragraph separators",
			input: "row one\u2028row two\u2029row three",
			want:  "row onerow tworow three",
		},
		{
			name:  "newline and tab",
			input: "line one\nline\ttwo",
			want:  "line onelinetwo",
		},
		{
			name:  "non-latin text is preserved",
			input: "スクリーンショット.png",
			want:  "スクリーンショット.png",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeTerminalText(tc.input)
			if got != tc.want {
				t.Fatalf("SanitizeTerminalText(%q) = %q, want %q", tc.input, got, tc.want)
			}
			if HasInterpretedTerminalSequence(got) {
				t.Fatalf("SanitizeTerminalText(%q) left interpreted sequences in %q", tc.input, got)
			}
		})
	}
}

func TestHasInterpretedTerminalSequenceDetectsControls(t *testing.T) {
	unsafe := []string{"\x1b[0m", "\u009b0m", "\u202e", "\u2069", "\u061c", "\u2028", "\u2029", "\x7f", "\r", "\n"}
	for _, value := range unsafe {
		if !HasInterpretedTerminalSequence(value) {
			t.Fatalf("HasInterpretedTerminalSequence(%q) = false, want true", value)
		}
	}

	safe := []string{"", "plain", "6.5-inch", "スクリーンショット"}
	for _, value := range safe {
		if HasInterpretedTerminalSequence(value) {
			t.Fatalf("HasInterpretedTerminalSequence(%q) = true, want false", value)
		}
	}
}
