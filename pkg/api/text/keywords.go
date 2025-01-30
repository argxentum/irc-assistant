package text

import (
	"slices"
	"strings"
)

// https://en.wikipedia.org/wiki/Most_common_words_in_English
var mostCommonEnglishWords = []string{
	"the", "be", "to", "of", "and", "a", "in", "that", "have", "I", "it", "for", "not", "on", "with", "he", "as", "you",
	"do", "at", "this", "but", "his", "by", "from", "they", "we", "say", "her", "she", "or", "an", "will", "my", "one",
	"all", "would", "there", "their", "what", "so", "up", "out", "if", "about", "who", "get", "which", "go", "me",
	"when", "make", "can", "like", "time", "no", "just", "him", "know", "take", "people", "into", "year", "your",
	"good", "some", "could", "them", "see", "other", "than", "then", "now", "look", "only", "come", "its", "over",
	"think", "also", "back", "after", "use", "two", "how", "our", "work", "first", "well", "way", "even", "new",
	"want", "because", "any", "these", "give", "day", "most", "us",
}

func ParseKeywords(quote string) []string {
	quote = strings.ToLower(quote)

	// remove any non-alphanumeric and non-space characters
	quote = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return -1
	}, quote)

	words := strings.Split(quote, " ")

	keywords := make([]string, 0)
	for _, word := range words {
		if !slices.Contains(mostCommonEnglishWords, word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func SanitizeKey(key string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, strings.ReplaceAll(strings.ToLower(strings.TrimSpace(key)), " ", "_"))
}
