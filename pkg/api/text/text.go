package text

import (
	"slices"
	"strings"
)

func Capitalize(s string, lowerRemainder bool) string {
	if len(s) == 0 {
		return s
	}

	result := strings.ToUpper(s[:1])
	if lowerRemainder {
		result += strings.ToLower(s[1:])
	} else {
		result += s[1:]
	}

	return result
}

func CapitalizeEveryWord(s string, lowerRemainder bool) string {
	words := strings.Fields(s)
	for i, word := range words {
		segments := strings.Split(word, "-")
		for j, segment := range segments {
			segments[j] = Capitalize(segment, lowerRemainder)
		}
		words[i] = strings.Join(segments, "-")
	}
	return strings.Join(words, " ")
}

func CapitalizeWords(s string, n int, lowerRemainder bool) string {
	words := strings.Fields(s)
	for i, word := range words {
		if i >= n {
			break
		}
		words[i] = Capitalize(word, lowerRemainder)
	}
	return strings.Join(words, " ")
}

func Uncapitalize(s string, lowerRemainder bool) string {
	if len(s) == 0 {
		return s
	}

	if mustBeCapitalized(s) {
		return Capitalize(s, lowerRemainder)
	}

	result := strings.ToLower(s[:1])
	if lowerRemainder {
		result += strings.ToLower(s[1:])
	} else {
		result += s[1:]
	}

	return result
}

func UncapitalizeWords(s string, n int, lowerRemainder bool) string {
	words := strings.Fields(s)
	for i, word := range words {
		if i >= n {
			break
		}
		words[i] = Uncapitalize(word, lowerRemainder)
	}
	return strings.Join(words, " ")
}

func mustBeCapitalized(s string) bool {
	if len(s) == 0 {
		return false
	}

	first := s[:1]
	if len(s) > 1 && strings.ToUpper(first) == "I" {
		second := s[1:2]
		if slices.Contains([]string{" ", "'"}, second) {
			return true
		}
	}

	return false
}

const defaultMaxLength = 256

func Sanitize(s string) string {
	return SanitizeToMaxLength(s, defaultMaxLength)
}

func SanitizeToMaxLength(s string, maxLength int) string {
	if len(s) == 0 {
		return s
	}

	// replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")

	// collapse multiple spaces
	s = strings.Join(strings.Fields(s), " ")

	// trim leading and trailing spaces
	s = strings.TrimSpace(s)

	// truncate to max length
	if len(s) > maxLength {
		s = s[:maxLength] + "..."
	}
	return s
}

func MostlyContains(a, b string, strength float32) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	a = strings.ToLower(a)
	b = strings.ToLower(b)

	if len(a) > len(b) {
		return MostlyContains(b, a, strength)
	}

	trim := int(float32(len(a)) * (1.0 - strength) * 0.5)
	start := trim
	end := len(a) - trim

	if start == end {
		return strings.Contains(b, a)
	}

	trimmed := strings.TrimSpace(a[start:end])

	if float32(len(trimmed))/float32(len(a)) < strength*0.75 {
		return false
	}

	return strings.Contains(b, trimmed)
}
