package promptutil

import (
	"encoding/json"
	"regexp"
	"strings"
)

var reCyrillicCheck = regexp.MustCompile(`[а-яА-ЯёЁ]`)

var reJunkLabels = regexp.MustCompile(`(?i)\b(BASE (POSITIVE|NEGATIVE) PROMPT|STYLE (NEGATIVE )?REFERENCE|USER (SCENE|DESCRIPTION)|USER NEGATIVE|MERGED PROMPT|NEGATIVE[_ ]PROMPT|Translation of non-English text|translates to|Merged Prompt)\s*:\s*`)
var reJSONFragments = regexp.MustCompile(`\{[^{}]*"(prompt|negative_prompt)"[^{}]*\}`)
var reQuotedStrings = regexp.MustCompile(`"[^"]{0,500}"`)
var reCyrillic = regexp.MustCompile(`[а-яА-ЯёЁ]+[^,(\[<]*,?`)

func ContainsCyrillic(s string) bool {
	return reCyrillicCheck.MatchString(s)
}

func ExtractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, `\_`, "_")
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	end := strings.LastIndex(s, "}")
	if end <= start {
		return s
	}
	return s[start : end+1]
}

func StripJunk(s string) string {
	if s == "" {
		return s
	}
	for reJunkLabels.MatchString(s) {
		s = reJunkLabels.ReplaceAllString(s, "")
	}
	for reJSONFragments.MatchString(s) {
		s = reJSONFragments.ReplaceAllString(s, "")
	}
	for strings.Contains(s, `"prompt"`) || strings.Contains(s, `"negative_prompt"`) {
		s = reQuotedStrings.ReplaceAllString(s, "")
	}
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.ReplaceAll(s, ", ,", ",")
	s = strings.ReplaceAll(s, ",,", ",")
	s = strings.Trim(s, " ,.\n\r")
	return s
}

func ExtractTagsFromRaw(raw string) string {
	var best string
	for _, m := range reQuotedStrings.FindAllString(raw, -1) {
		m = strings.Trim(m, `"`)
		m = strings.TrimSpace(m)
		if len(m) > len(best) && !strings.Contains(m, `"`) && !ContainsCyrillic(m) && (strings.Contains(m, ", ") || strings.Contains(m, "quality")) {
			best = m
		}
	}
	return best
}

func ExtractNegativeFromRaw(raw string) string {
	jsonRaw := ExtractJSON(raw)
	if jsonRaw == "" {
		return ""
	}
	var obj map[string]string
	if err := json.Unmarshal([]byte(jsonRaw), &obj); err != nil {
		return ""
	}
	if np, ok := obj["negative_prompt"]; ok && !ContainsCyrillic(np) {
		return np
	}
	return ""
}

func TruncateRepetitive(s string, maxLen int) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, ", ")
	result := make([]string, 0, len(parts))
	prevPrefix := ""
	repeatCount := 0
	for _, part := range parts {
		prefix := part
		if idx := strings.Index(part, ":"); idx > 0 {
			prefix = part[:idx]
		}
		prefix = strings.ToLower(strings.TrimSpace(prefix))
		if prefix == prevPrefix && prefix != "" {
			repeatCount++
			if repeatCount >= 3 {
				break
			}
		} else {
			prevPrefix = prefix
			repeatCount = 0
		}
		result = append(result, part)
	}
	s = strings.Join(result, ", ")
	if len(s) > maxLen {
		if idx := strings.LastIndex(s[:maxLen], ","); idx > 0 {
			s = s[:idx]
		} else {
			s = s[:maxLen]
		}
	}
	s = strings.TrimRight(s, " ,.")
	return s
}

func SplitCompositeSampler(sampler, scheduleType string) (string, string) {
	if scheduleType != "" {
		return sampler, scheduleType
	}
	knownSchedulers := []string{"Karras", "Exponential", "Polyexponential"}
	for _, s := range knownSchedulers {
		if strings.HasSuffix(sampler, " "+s) {
			return sampler[:len(sampler)-len(s)-1], s
		}
	}
	return sampler, ""
}

func BuildSamplerName(sampler, scheduleType string) string {
	if scheduleType != "" {
		st := strings.ToUpper(scheduleType[:1]) + scheduleType[1:]
		return sampler + " " + st
	}
	return sampler
}

func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
