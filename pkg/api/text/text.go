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
