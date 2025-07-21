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
	Headers     map[string]string
}

var DefaultRetrievalParams = RetrievalParams{
	Method:      http.MethodGet,
	Retries:     DefaultRetries,
	RetryDelay:  DefaultRetryDelay,
	Timeout:     DefaultTimeout,
	Impersonate: true,
	Headers:     make(map[string]string),
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
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":           "en-US,en;q=0.9,eo;q=0.8",
		"Cache-Control":             "max-age=0",
		"Priority":                  "u=0, i",
		"Sec-Ch-Ua":                 `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`,
		"Sec-Ch-Ua-Mobile":          `?0`,
		"Sec-Ch-Ua-Platform":        "Windows",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
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

func RandomHeaderSet() map[string]string {
	return headerSets[rand.IntN(len(headerSets))]
}

func Domain(url string) string {
	matches := domainRegexp.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return url
}

type RequestLabeler struct {
	req *http.Request
}

func NewRequestLabeler(req *http.Request) *RequestLabeler {
	return &RequestLabeler{req: req}
}

func (rl *RequestLabeler) Labels() map[string]string {
	labels := make(map[string]string)
	labels["url"] = rl.req.URL.String()
	labels["method"] = rl.req.Method
	for k, vs := range rl.req.Header {
		labels[k] = vs[0]
	}
	return labels
}

type ResponseLabeler struct {
	resp *http.Response
}

func NewResponseLabeler(resp *http.Response) *ResponseLabeler {
	return &ResponseLabeler{resp: resp}
}

func (rl *ResponseLabeler) Labels() map[string]string {
	labels := make(map[string]string)
	labels["url"] = rl.resp.Request.URL.String()
	labels["method"] = rl.resp.Request.Method
	labels["status"] = rl.resp.Status
	for k, vs := range rl.resp.Header {
		labels[k] = vs[0]
	}
	return labels
}
