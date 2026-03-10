package tdt

import (
	"strings"
	"unicode"
)

// tokenize splits text into lowercase tokens, splitting on whitespace,
// punctuation, underscores, and camelCase boundaries.
// Tokens shorter than 2 characters are dropped.
func tokenize(text string) []string {
	if text == "" {
		return nil
	}
	// Split camelCase on the original text first, then process.
	expanded := splitCamelCase(text)
	// Replace underscores with spaces.
	expanded = strings.ReplaceAll(expanded, "_", " ")
	// Split on non-letter, non-digit characters.
	parts := strings.FieldsFunc(expanded, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	// Lowercase and filter short tokens.
	var tokens []string
	for _, p := range parts {
		p = strings.ToLower(p)
		if len(p) >= 2 {
			tokens = append(tokens, p)
		}
	}
	return tokens
}

// splitCamelCase inserts spaces before uppercase letters that follow lowercase
// letters, e.g. "getWeather" -> "get Weather".
func splitCamelCase(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
