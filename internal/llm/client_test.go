package llm

import "testing"

func TestTruncateRepetitive(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "empty string",
			input:  "",
			maxLen: 1000,
			want:   "",
		},
		{
			name:   "normal prompt unchanged",
			input:  "masterpiece, best quality, detailed skin, beautiful eyes, long hair",
			maxLen: 1000,
			want:   "masterpiece, best quality, detailed skin, beautiful eyes, long hair",
		},
		{
			name:   "truncate repetitive blouse pattern",
			input:  "masterpiece, best quality, (blouse:1), (blouse:open), (blouse:satin), (blouse:glitter), (blouse:blouse), (blouse:blouse:blouse), (blouse:blouse:blouse:blouse)",
			maxLen: 1000,
			want:   "masterpiece, best quality, (blouse:1), (blouse:open), (blouse:satin)",
		},
		{
			name:   "allow up to 3 same-prefix tags",
			input:  "(hair:blonde:1.2), (hair:long:1.1), (hair:wavy:1.0)",
			maxLen: 1000,
			want:   "(hair:blonde:1.2), (hair:long:1.1), (hair:wavy:1.0)",
		},
		{
			name:   "hard truncation at maxLen",
			input:  "masterpiece, best quality, " + string(make([]byte, 2000)),
			maxLen: 50,
			want:   "masterpiece, best quality",
		},
		{
			name:   "different prefixes not truncated",
			input:  "masterpiece, best quality, (eyes:blue:1.2), (hair:blonde:1.1), (skin:detailed:1.0)",
			maxLen: 1000,
			want:   "masterpiece, best quality, (eyes:blue:1.2), (hair:blonde:1.1), (skin:detailed:1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateRepetitive(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateRepetitive() = %q, want %q", got, tt.want)
			}
		})
	}
}
