package retriever

import (
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const DefaultRetries = 5
const DefaultRetryDelay = 150
const DefaultTimeout = 1500

var NoResponseError = errors.New("no content")
var DisallowedContentTypeError = errors.New("disallowed content type")
var RequestTimedOutError = errors.New("request timed out")

var retryDelayOffset = time.Duration(100)

type RetrievalParams struct {
	Method      string
	URL         string
	Body        io.Reader
	Retries     int
	RetryDelay  int
	Timeout     time.Duration
	Impersonate bool
}

var DefaultRetrievalParams = RetrievalParams{
	Method:      http.MethodGet,
	Retries:     DefaultRetries,
	RetryDelay:  DefaultRetryDelay,
	Timeout:     DefaultTimeout,
	Impersonate: true,
}

func DefaultParams(url string) RetrievalParams {
	params := DefaultRetrievalParams
	params.URL = url
	return params
}

type retrieved struct {
	response *http.Response
	err      error
}

var domainRegexp = regexp.MustCompile(`https?://([^/]+)`)
var rootDomainRegexp = regexp.MustCompile(`https?://.*?([^.]+\.[a-z]+)(?:/|$)`)

var headerSets = []map[string]string{
	{
		"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language": "en-US,en;q=0.9",
	},
	{
		"User-Agent":         "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		"Accept":             "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":    "en-US,en;q=0.9,eo;q=0.8",
		"sec-ch-ua":          `"Not(A:Brand";v="99", "Google Chrome";v="133", "Chromium";v="133"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": `"Linux"`,
		"sec-fetch-dest":     "document",
		"sec-fetch-mode":     "navigate",
		"sec-fetch-site":     "none",
		"sec-fetch-user":     "?1",
	},
}

var allowedContentTypePrefixes = []string{
	"text/html",
	"text/plain",
	"text/xml",
	"application/xml",
	"application/xhtml",
	"application/rss",
	"application/atom",
	"application/rdf",
	"application/json",
	"application/ld+json",
	"application/vnd.api",
	"application/hal+json",
	"application/vnd.collection",
}

func IsContentTypeAllowed(contentType string) bool {
	if contentType == "" {
		return true
	}
	for _, p := range allowedContentTypePrefixes {
		if strings.HasPrefix(contentType, p) {
			return true
		}
	}
	return false
}

func translateURL(url string) string {
	domain := RootDomain(url)
	switch domain {
	case "x.com":
		return replaceRoot(url, "x.com", "fixupx.com")
	case "twitter.com":
		return replaceRoot(url, "twitter.com", "fxtwitter.com")
	}
	return url
}

func replaceRoot(url, old, new string) string {
	return strings.Replace(url, old, new, 1)
}

func RandomHeaderSet() map[string]string {
	return headerSets[rand.IntN(len(headerSets))]
}

func RootDomain(url string) string {
	matches := rootDomainRegexp.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return url
}

func Domain(url string) string {
	matches := domainRegexp.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return url
}
