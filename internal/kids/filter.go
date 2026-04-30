package kids

import (
	"fmt"
	"regexp"
	"strings"
)

type Category struct {
	Name        string
	Label       string
	Words       []string
	AlwaysOn    bool
	NegativeTag string
}

var Categories = []Category{
	{
		Name:     "nsfw",
		Label:    "NSFW / Adult",
		AlwaysOn: true,
		Words: []string{
			"nsfw", "nude", "naked", "porn", "pornography", "erotic", "sexual", "sexy",
			"boobs", "breasts", "nipples", "genitals", "penis", "vagina", "anus",
			"sex", "orgasm", "fetish", "bondage", "bdsm", "hentai",
		},
		NegativeTag: "nsfw, nude, naked, porn, erotic, sexual",
	},
	{
		Name:     "selfharm",
		Label:    "Self-harm & Hate",
		AlwaysOn: true,
		Words: []string{
			"suicide", "self-harm", "selfharm", "cutting",
			"racism", "racist", "hate speech", "nazi", "swastika",
		},
		NegativeTag: "self-harm, suicide, hate speech",
	},
	{
		Name:  "violence",
		Label: "Violence",
		Words: []string{
			"violence", "violent", "gore", "gory", "blood", "bloody", "bloode",
			"murder", "kill", "killing", "dead", "death", "corpse",
			"torture", "tortured", "abuse", "abused", "assault", "raped", "rape",
		},
		NegativeTag: "violence, gore, blood, torture, death, kill, murder, abuse, assault",
	},
	{
		Name:  "horror",
		Label: "Horror & Scary",
		Words: []string{
			"horror", "terrifying", "frightening", "creepy", "disturbing", "scary",
			"zombie", "zombies", "monster", "demons", "demon",
		},
		NegativeTag: "horror, disturbing, frightening, scary, creepy",
	},
	{
		Name:  "weapons",
		Label: "Weapons",
		Words: []string{
			"weapon", "gun", "guns", "pistol", "rifle", "knife",
		},
		NegativeTag: "weapon harm",
	},
	{
		Name:  "substances",
		Label: "Substances",
		Words: []string{
			"drugs", "drug", "cocaine", "heroin", "meth", "marijuana", "weed",
			"alcohol", "drunk", "smoking", "cigarette", "cigar",
		},
		NegativeTag: "drugs, alcohol, smoking",
	},
	{
		Name:  "mature",
		Label: "General Mature",
		Words: []string{
			"explicit", "mature", "xxx", "obscene", "obscenity", "profanity",
		},
		NegativeTag: "mature content, explicit, obscene",
	},
}

var safeDefault = "safe landscape, beautiful nature, sunny day, clear sky, peaceful meadow, colorful flowers, butterflies, gentle breeze, warm sunlight, family-friendly, wholesome, cute animals"

func buildRegex(disabledCategories map[string]bool) *regexp.Regexp {
	var words []string
	for _, cat := range Categories {
		if cat.AlwaysOn || !disabledCategories[cat.Name] {
			words = append(words, cat.Words...)
		}
	}
	if len(words) == 0 {
		return nil
	}
	patterns := make([]string, len(words))
	for i, w := range words {
		patterns[i] = `\b` + regexp.QuoteMeta(w) + `\b`
	}
	return regexp.MustCompile("(?i)(" + strings.Join(patterns, "|") + ")")
}

func FilterInput(desc string, disabledCategories map[string]bool) (string, error) {
	re := buildRegex(disabledCategories)
	if re != nil && re.MatchString(desc) {
		return "", fmt.Errorf("content blocked by Kids Mode safety filter")
	}
	return desc, nil
}

func FilterOutput(prompt string, disabledCategories map[string]bool) string {
	re := buildRegex(disabledCategories)
	if re == nil {
		return prompt
	}
	parts := strings.Split(prompt, ",")
	var filtered []string
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		if re.MatchString(tag) {
			continue
		}
		filtered = append(filtered, tag)
	}
	if len(filtered) < 3 {
		return safeDefault
	}
	return strings.Join(filtered, ", ")
}

func NegativePrompt(disabledCategories map[string]bool) string {
	var parts []string
	for _, cat := range Categories {
		if cat.AlwaysOn || !disabledCategories[cat.Name] {
			if cat.NegativeTag != "" {
				parts = append(parts, cat.NegativeTag)
			}
		}
	}
	return strings.Join(parts, ", ")
}
