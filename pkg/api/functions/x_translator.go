package functions

import (
	"regexp"
	"strings"
)

type XTranslator interface {
	TranslateURL(url string) (string, bool)
}

func NewXTranslator() XTranslator {
	return &xTranslator{
		rootRegexp: regexp.MustCompile(`https?://.*?([^.]+\.[a-z]+)(?:/|$)`),
	}
}

type xTranslator struct {
	rootRegexp *regexp.Regexp
}

func (t *xTranslator) TranslateURL(url string) (string, bool) {
	domain := t.getRootDomain(url)
	switch domain {
	case "x.com":
		return replaceRoot(url, "x.com", "fixupx.com"), true
	case "twitter.com":
		return replaceRoot(url, "twitter.com", "fxtwitter.com"), true
	}
	return url, false
}

func (t *xTranslator) getRootDomain(url string) string {
	matches := t.rootRegexp.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return url
}

func replaceRoot(url, old, new string) string {
	return strings.Replace(url, old, new, 1)
}
