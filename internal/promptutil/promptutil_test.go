package promptutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsCyrillic_HasCyrillic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"russian word", "привет", true},
		{"mixed latin and cyrillic", "hello мир", true},
		{"cyrillic with numbers", "тест123", true},
		{"uppercase cyrillic", "МОСКВА", true},
		{"yo letter", "ёж", true},
		{"uppercase yo", "Ёлка", true},
		{"pure latin", "hello world", false},
		{"numbers only", "12345", false},
		{"empty string", "", false},
		{"special chars only", "!@#$%", false},
		{"english with numbers", "test42", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ContainsCyrillic(tt.input))
		})
	}
}

func TestExtractJSON_CleansAndExtracts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"clean json object", `{"prompt": "cat"}`, `{"prompt": "cat"}`},
		{"json in markdown block", "```json\n{\"prompt\": \"cat\"}\n```", `{"prompt": "cat"}`},
		{"json in plain code block", "```\n{\"prompt\": \"cat\"}\n```", `{"prompt": "cat"}`},
		{"json with surrounding text", `Here is the result: {"prompt": "cat"} done`, `{"prompt": "cat"}`},
		{"escaped underscores in content", `{"prompt": "cat\_dog"}`, `{"prompt": "cat_dog"}`},
		{"whitespace padded", `  {"prompt": "cat"}  `, `{"prompt": "cat"}`},
		{"no braces", "just plain text", "just plain text"},
		{"only opening brace", `{"prompt": "cat"`, `{"prompt": "cat"`},
		{"only closing brace", `"prompt": "cat"}`, `"prompt": "cat"}`},
		{"empty string", "", ""},
		{"nested braces", `outer {"inner": "val"} text`, `{"inner": "val"}`},
		{"reversed braces", `} {`, `} {`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ExtractJSON(tt.input))
		})
	}
}

func TestStripJunk_RemovesLabelsAndFragments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"base positive prompt label", "BASE POSITIVE PROMPT: a cat, best quality", "a cat, best quality"},
		{"base negative prompt label", "BASE NEGATIVE PROMPT: ugly, blurry", "ugly, blurry"},
		{"user description label", "USER DESCRIPTION: a beautiful landscape", "a beautiful landscape"},
		{"merged prompt label", "MERGED PROMPT: forest, trees", "forest, trees"},
		{"negative prompt label with underscore", "NEGATIVE_PROMPT: bad quality", "bad quality"},
		{"negative prompt label with space", "NEGATIVE PROMPT: bad quality", "bad quality"},
		{"json fragment with prompt key", `extra {"prompt": "cat, dog"} end`, "extra end"},
		{"json fragment with negative_prompt key", `extra {"negative_prompt": "bad"} end`, "extra end"},
		{"double spaces", "a  b  c", "a b c"},
		{"double commas", "a,, b", "a, b"},
		{"comma space comma", "a, , b", "a, b"},
		{"trailing spaces and commas", "  a, b, ", "a, b"},
		{"multiple junk labels", "BASE POSITIVE PROMPT: MERGED PROMPT: cat", "cat"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, StripJunk(tt.input))
		})
	}
}

func TestExtractTagsFromRaw_FindsBestQuotedString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"single quoted with comma", `{"prompt": "cat, best quality"}`, "cat, best quality"},
		{"multiple quoted picks longest", `"short" and "longer tag, high quality, detailed"`, "longer tag, high quality, detailed"},
		{"no quoted strings", "just plain text", ""},
		{"empty input", "", ""},
		{"skips cyrillic content", `"привет мир, test, quality"`, ""},
		{"skips without comma or quality", `"single_tag"`, ""},
		{"contains quality keyword", `"best quality artwork"`, "best quality artwork"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ExtractTagsFromRaw(tt.input))
		})
	}
}

func TestExtractNegativeFromRaw_ExtractsFromJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid json with negative_prompt", `{"prompt": "cat", "negative_prompt": "ugly, blurry"}`, "ugly, blurry"},
		{"markdown wrapped", "```json\n{\"prompt\": \"cat\", \"negative_prompt\": \"bad\"}\n```", "bad"},
		{"no negative_prompt key", `{"prompt": "cat"}`, ""},
		{"cyrillic negative_prompt skipped", `{"prompt": "cat", "negative_prompt": "плохо, размыто"}`, ""},
		{"invalid json", "not json at all", ""},
		{"empty input", "", ""},
		{"negative_prompt empty string", `{"prompt": "cat", "negative_prompt": ""}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ExtractNegativeFromRaw(tt.input))
		})
	}
}

func TestTruncateRepetitive_StopsRepetition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"empty string", "", 100, ""},
		{"no repetition", "cat, dog, bird", 100, "cat, dog, bird"},
		{"repeated prefix stops at 3", "girl: a, girl: b, girl: c, girl: d, girl: e", 100, "girl: a, girl: b, girl: c"},
		{"different prefixes pass through", "cat: 1, dog: 2, cat: 3, dog: 4", 100, "cat: 1, dog: 2, cat: 3, dog: 4"},
		{"truncates at maxLen without comma in range", "a very long string that exceeds, the maximum allowed length", 30, "a very long string that exceed"},
		{"truncates at maxLen without comma", "abcdefghijklmnopqrstuvwxyz", 10, "abcdefghij"},
		{"maxLen larger than string", "short", 100, "short"},
		{"single item no comma", "single_tag", 100, "single_tag"},
		{"trailing cleanup", "cat, dog, ", 100, "cat, dog"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, TruncateRepetitive(tt.input, tt.maxLen))
		})
	}
}

func TestSplitCompositeSampler_SplitsWhenComposite(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		sampler         string
		scheduleType    string
		wantSampler     string
		wantSchedule    string
	}{
		{"sampler with Karras", "Euler a Karras", "", "Euler a", "Karras"},
		{"sampler with Exponential", "DPM++ 2M Exponential", "", "DPM++ 2M", "Exponential"},
		{"sampler with Polyexponential", "DPM++ SDE Polyexponential", "", "DPM++ SDE", "Polyexponential"},
		{"plain sampler no schedule", "Euler a", "", "Euler a", ""},
		{"schedule type already set", "Euler a", "Karras", "Euler a", "Karras"},
		{"empty sampler", "", "", "", ""},
		{"empty schedule type override", "Euler", "Automatic", "Euler", "Automatic"},
		{"Karras in middle not split", "Karras Euler", "", "Karras Euler", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sampler, schedule := SplitCompositeSampler(tt.sampler, tt.scheduleType)
			assert.Equal(t, tt.wantSampler, sampler)
			assert.Equal(t, tt.wantSchedule, schedule)
		})
	}
}

func TestTruncate_TruncatesWithEllipsis(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		n     int
		want  string
	}{
		{"short string unchanged", "cat", 10, "cat"},
		{"exact length unchanged", "cat", 3, "cat"},
		{"long string truncated", "hello world", 5, "hello..."},
		{"empty string", "", 5, ""},
		{"zero n", "hello", 0, "..."},
		{"single char n", "hello", 1, "h..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Truncate(tt.input, tt.n))
		})
	}
}
