package utils

import (
	"strings"
	"unicode"
)

func CountStoryUnits(text string) int {
	count := 0
	inLatinToken := false
	for _, r := range strings.TrimSpace(text) {
		switch {
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r):
			if inLatinToken {
				inLatinToken = false
			}
			count++
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if !inLatinToken {
				count++
				inLatinToken = true
			}
		default:
			inLatinToken = false
		}
	}
	return count
}
