package kids

import (
	"strings"
)

var blocklist = []string{
	"nsfw", "nude", "naked", "porn", "pornography", "erotic", "sexual", "sexy",
	"boobs", "breasts", "nipples", "genitals", "penis", "vagina", "anus",
	"sex", "orgasm", "fetish", "bondage", "bdsm", "hentai",
	"violence", "violent", "gore", "gory", "blood", "bloody", "bloode",
	"murder", "kill", "killing", "dead", "death", "corpse", "dead body",
	"torture", "tortured", "abuse", "abused", "assault", "raped", "rape",
	"horror", "terrifying", "frightening", "creepy", "disturbing", "scary",
	"zombie", "zombies", "monster", "demons", "demon",
	"drugs", "drug", "cocaine", "heroin", "meth", "marijuana", "weed",
	"alcohol", "drunk", "smoking", "cigarette", "cigar",
	"suicide", "self-harm", "selfharm", "cutting",
	"weapon", "gun", "guns", "pistol", "rifle", "knife", "sword fight",
	"racism", "racist", "hate", "hate speech", "nazi", "swastika",
	"explicit", "mature", "xxx", "obscene", "obscenity", "profanity",
}

var safeDefault = "safe landscape, beautiful nature, sunny day, clear sky, peaceful meadow, colorful flowers, butterflies, gentle breeze, warm sunlight, family-friendly, wholesome, cute animals"

func FilterInput(desc string) string {
	lower := strings.ToLower(desc)
	for _, w := range blocklist {
		if strings.Contains(lower, w) {
			return ""
		}
	}
	return desc
}

func FilterOutput(prompt string) string {
	parts := strings.Split(prompt, ",")
	var filtered []string
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		lower := strings.ToLower(tag)
		blocked := false
		for _, w := range blocklist {
			if strings.Contains(lower, w) {
				blocked = true
				break
			}
		}
		if !blocked {
			filtered = append(filtered, tag)
		}
	}
	if len(filtered) < 3 {
		return safeDefault
	}
	return strings.Join(filtered, ", ")
}
