package retriever

import (
	"errors"
	"github.com/bobesa/go-domain-util/domainutil"
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

func (params RetrievalParams) WithTimeout(timeout time.Duration) RetrievalParams {
	params.Timeout = timeout
	return params
}

func (params RetrievalParams) WithImpersonation(impersonation bool) RetrievalParams {
	params.Impersonate = impersonation
	return params
}

type retrieved struct {
	response *http.Response
	err      error
}

var domainRegexp = regexp.MustCompile(`https?://([^/]+)`)

var headerSets = []map[string]string{
	{
		"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language": "en-US,en;q=0.9",
	},
	{
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":           "en-US,en;q=0.9",
		"Cookie":                    "MUID=17487EC1D02B66F71ED86B78D11F6764; MUIDB=17487EC1D02B66F71ED86B78D11F6764; _EDGE_S=F=1&SID=343961E3431468C313ED745A422069F1; _EDGE_V=1; SRCHD=AF=NOFORM; SRCHUID=V=2&GUID=7CEB4E628FEE4938BCA767E361ACADD8&dmnchg=1; SRCHUSR=DOB=20250324; _SS=SID=343961E3431468C313ED745A422069F1; ak_bmsc=5C12A21850150D7CA36EE3E4BD0ADB1A~000000000000000000000000000000~YAAQCajOFw/lHKqVAQAACi17yhuUMb80e4F2Es+NOo89hsGywq6tMkiibL7LJU3ePgwQzzzIECVZTsLybCCLmkvCkwITIZjVBfulzDAyvtJLbxKe0aKmVHhlJimN/EF3i0TqiHC2yKuDHA/JQYQgycnM3k+iKqa5pZnr7lTYXKqhpPKERY/sUZ95g3I+xR0uDeCTLe0y3H0UjayUN/z2QyWtVcsJ9auyyLJtMdwFHiYSI+j61MkQ0CasvaQtJ2XY0ZN9L5LRM2evjJFlbC8KFt2BWqi6frL5lRBbZSoQkO6AOpKEstNSnI70tmofmpqa+bohBVurbFIjp3NVOXbcsS2OAuxUJFaQQtfgJsVowJY+xqrj5BEe0dkQ+f81YQ==; SRCHHPGUSR=SRCHLANG=en&IG=026F74EF480F4457A9CF099DA04EA89F; bm_sv=05279D4C2401A6EA4B61FB7F51729605~YAAQG6jOF6jdBqiVAQAAG9KlyhsJCYgcb2KZCxMyrB/GU5OZuXA7OIH2vApvs2qX6vDzhaKZjEingtYLCJb9g2MVcCuPrjjQc5PZIQxA501WcH20ZkmFLa4zp+ec6hVjhAtgmTeFladuSW54S6J9GMs+wP0pL5ZII6ovLse/TFh6KACZ+pyRx6hY2InjEbiGOXXcvLTw2J1WWSgNxF8rB3sbG8UDMC5h7gx6sIun+5BAiq+AR7jfWyLzijTxkA==~1",
		"Dnt":                       "1",
		"Priority":                  "u=0, i",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
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
	return domainutil.Domain(url)
}

func Domain(url string) string {
	matches := domainRegexp.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return url
}
