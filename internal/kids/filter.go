package kids

import (
	"fmt"
	"regexp"
	"strings"
)

var blocklist = []string{
	"nsfw", "nude", "naked", "porn", "pornography", "erotic", "sexual", "sexy",
	"boobs", "breasts", "nipples", "genitals", "penis", "vagina", "anus",
	"sex", "orgasm", "fetish", "bondage", "bdsm", "hentai",
	"violence", "violent", "gore", "gory", "blood", "bloody", "bloode",
	"murder", "kill", "killing", "dead", "death", "corpse",
	"torture", "tortured", "abuse", "abused", "assault", "raped", "rape",
	"horror", "terrifying", "frightening", "creepy", "disturbing", "scary",
	"zombie", "zombies", "monster", "demons", "demon",
	"drugs", "drug", "cocaine", "heroin", "meth", "marijuana", "weed",
	"alcohol", "drunk", "smoking", "cigarette", "cigar",
	"suicide", "self-harm", "selfharm", "cutting",
	"weapon", "gun", "guns", "pistol", "rifle", "knife",
	"racism", "racist", "hate speech", "nazi", "swastika",
	"explicit", "mature", "xxx", "obscene", "obscenity", "profanity",
}

var blockRegex *regexp.Regexp

func init() {
	patterns := make([]string, len(blocklist))
	for i, w := range blocklist {
		escaped := regexp.QuoteMeta(w)
		patterns[i] = `\b` + escaped + `\b`
	}
	blockRegex = regexp.MustCompile("(?i)(" + strings.Join(patterns, "|") + ")")
}

var safeDefault = "safe landscape, beautiful nature, sunny day, clear sky, peaceful meadow, colorful flowers, butterflies, gentle breeze, warm sunlight, family-friendly, wholesome, cute animals"

func FilterInput(desc string) (string, error) {
	if blockRegex.MatchString(desc) {
		return "", fmt.Errorf("content blocked by Kids Mode safety filter")
	}
	return desc, nil
}

func FilterOutput(prompt string) string {
	parts := strings.Split(prompt, ",")
	var filtered []string
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		if blockRegex.MatchString(tag) {
			continue
		}
		filtered = append(filtered, tag)
	}
	if len(filtered) < 3 {
		return safeDefault
	}
	return strings.Join(filtered, ", ")
}
