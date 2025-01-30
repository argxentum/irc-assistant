package text

import "strings"

func ContainsAny(s string, chars []string) bool {
	for _, c := range chars {
		if strings.Contains(s, c) {
			return true
		}
	}
	return false
}

func ContainsAll(s string, chars []string) bool {
	for _, c := range chars {
		if !strings.Contains(s, c) {
			return false
		}
	}
	return true
}
