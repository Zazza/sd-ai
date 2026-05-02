package kids

import (
	"strings"
	"testing"
)

func TestFilterInput_FalsePositives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "class should not match ass",
			input: "a beautiful classroom with students",
		},
		{
			name:  "dismissing should not match smoking",
			input: "dismissing the idea quickly",
		},
		{
			name:  "assignment should not match ass",
			input: "homework assignment on the desk",
		},
		{
			name:  "classic should not match ass",
			input: "classic vintage car parked outside",
		},
		{
			name:  "assume should not match ass",
			input: "we can assume the weather is nice",
		},
		{
			name:  "assemble should not match ass",
			input: "assemble the puzzle pieces together",
		},
		{
			name:  "session should not match ass",
			input: "a study session at the library",
		},
		{
			name:  "mastery should not match ass",
			input: "showing mastery of the subject",
		},
		{
			name:  "compassion should not match ass",
			input: "showing compassion to others",
		},
		{
			name:  "bass guitar should not match ass",
			input: "playing bass guitar on stage",
		},
		{
			name:  "assassin should not match ass standalone",
			input: "an assassin in the shadows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := FilterInput(tt.input, nil)
			if err != nil {
				t.Errorf("FilterInput(%q) false positive block: %v", tt.input, err)
			}
			if result != tt.input {
				t.Errorf("FilterInput(%q) = %q, want %q", tt.input, result, tt.input)
			}
		})
	}
}

func TestFilterInput_BlockedContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "blocked word nude",
			input: "a nude painting",
		},
		{
			name:  "blocked word violence uppercase",
			input: "VIOLENCE in the scene",
		},
		{
			name:  "blocked word gore",
			input: "gore and blood everywhere",
		},
		{
			name:  "blocked word drugs",
			input: "taking drugs at a party",
		},
		{
			name:  "blocked word murder",
			input: "a murder mystery novel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := FilterInput(tt.input, nil)
			if err == nil {
				t.Errorf("FilterInput(%q) expected block, got pass", tt.input)
			}
			if err.Error() != "content blocked by Kids Mode safety filter" {
				t.Errorf("FilterInput error = %q, want %q", err.Error(), "content blocked by Kids Mode safety filter")
			}
		})
	}
}

func TestFilterInput_SafeContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "nature description",
			input: "a beautiful sunset over the mountains with flowers",
		},
		{
			name:  "portrait description",
			input: "a smiling girl with blue eyes and long dress",
		},
		{
			name:  "empty string",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := FilterInput(tt.input, nil)
			if err != nil {
				t.Errorf("FilterInput(%q) unexpected error: %v", tt.input, err)
			}
			if result != tt.input {
				t.Errorf("FilterInput(%q) = %q, want %q", tt.input, result, tt.input)
			}
		})
	}
}

func TestFilterInput_DisabledCategories(t *testing.T) {
	t.Parallel()

	disabled := map[string]bool{
		"violence":   true,
		"horror":     true,
		"weapons":    true,
		"substances": true,
		"mature":     true,
	}

	_, err := FilterInput("violence in the scene", disabled)
	if err != nil {
		t.Errorf("FilterInput with violence disabled should pass, got error: %v", err)
	}

	_, err = FilterInput("nude painting", disabled)
	if err == nil {
		t.Error("FilterInput with nsfw always-on should block nude")
	}
}

func TestFilterOutput_RemovesBlockedTags(t *testing.T) {
	t.Parallel()

	input := "beautiful eyes, nude, long hair, violence, red dress"
	result := FilterOutput(input, nil)

	if strings.Contains(result, "nude") {
		t.Errorf("FilterOutput() should remove 'nude', got %q", result)
	}
	if strings.Contains(result, "violence") {
		t.Errorf("FilterOutput() should remove 'violence', got %q", result)
	}
	if !strings.Contains(result, "beautiful eyes") {
		t.Errorf("FilterOutput() should keep 'beautiful eyes', got %q", result)
	}
	if !strings.Contains(result, "red dress") {
		t.Errorf("FilterOutput() should keep 'red dress', got %q", result)
	}
}

func TestFilterOutput_TooFewTagsReturnsSafe(t *testing.T) {
	t.Parallel()

	input := "nude, violence, murder"
	result := FilterOutput(input, nil)

	if result != safeDefault {
		t.Errorf("FilterOutput() with all blocked tags = %q, want %q", result, safeDefault)
	}
}

func TestFilterOutput_AllSafe(t *testing.T) {
	t.Parallel()

	input := "blue eyes, long hair, red dress, outdoors, sunny"
	result := FilterOutput(input, nil)

	want := "blue eyes, long hair, red dress, outdoors, sunny"
	if result != want {
		t.Errorf("FilterOutput() = %q, want %q", result, want)
	}
}

func TestNegativePrompt(t *testing.T) {
	t.Parallel()

	result := NegativePrompt(nil)

	if !strings.Contains(result, "nsfw") {
		t.Errorf("NegativePrompt() should contain nsfw tags, got %q", result)
	}
	if !strings.Contains(result, "self-harm") {
		t.Errorf("NegativePrompt() should contain self-harm tags, got %q", result)
	}
	if !strings.Contains(result, "violence") {
		t.Errorf("NegativePrompt() should contain violence tags, got %q", result)
	}
}

func TestNegativePrompt_DisabledCategories(t *testing.T) {
	t.Parallel()

	disabled := map[string]bool{
		"violence":   true,
		"horror":     true,
		"weapons":    true,
		"substances": true,
		"mature":     true,
	}

	result := NegativePrompt(disabled)

	if strings.Contains(result, "violence") {
		t.Errorf("NegativePrompt() with violence disabled should not contain violence, got %q", result)
	}
	if !strings.Contains(result, "nsfw") {
		t.Errorf("NegativePrompt() should always contain nsfw, got %q", result)
	}
}
