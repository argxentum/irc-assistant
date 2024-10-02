package retriever

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"io"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var headerSets = []map[string]string{
	{
		"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language": "en-US,en;q=0.9",
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

var rootDomainRegexp = regexp.MustCompile(`https?://.*?([^.]+\.[a-z]+)(?:/|$)`)

const DefaultRetries = 5
const DefaultRetryDelay = 150
const DefaultTimeout = 1500

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

type DocumentRetriever interface {
	RetrieveDocument(e *irc.Event, params RetrievalParams, timeout time.Duration) (*goquery.Document, error)
	RetrieveDocumentSelection(e *irc.Event, params RetrievalParams, selector string) (*goquery.Selection, error)
	Parse(e *irc.Event, doc *goquery.Document, selectors ...string) []*goquery.Selection
}

func NewDocumentRetriever() DocumentRetriever {
	return &retriever{
		//
	}
}

type retriever struct {
	//
}

type result struct {
	doc *goquery.Selection
	err error
}

var DisallowedContentTypeError = errors.New("disallowed content type")
var RequestTimedOutError = errors.New("request timed out")

type retrieved struct {
	response *http.Response
	err      error
}

func (r *retriever) RetrieveDocument(e *irc.Event, params RetrievalParams, timeout time.Duration) (*goquery.Document, error) {
	logger := log.Logger()

	translated := translateURL(params.URL)
	if translated != params.URL {
		logger.Debugf(e, "translated %s to %s", params.URL, translated)
	}
	params.URL = translated

	req, err := http.NewRequest(params.Method, params.URL, params.Body)
	if err != nil {
		logger.Debugf(e, "request creation error, %s", err)
		return nil, err
	}

	if params.Impersonate {
		logger.Debugf(e, "adding impersonation request headers")
		headers := RandomHeaderSet()
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	var rc = make(chan retrieved)
	go func() {
		go func() {
			time.Sleep(timeout * time.Millisecond)
			logger.Debugf(e, "timing out request")
			rc <- retrieved{nil, RequestTimedOutError}
		}()

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Debugf(e, "retrieval error (status %d), %s", resp.StatusCode, err)
			rc <- retrieved{nil, err}
		}
		if resp == nil {
			logger.Debugf(e, "retrieval error (status %s)", resp.Status)
			rc <- retrieved{nil, errors.New("no response")}
		}
		rc <- retrieved{resp, nil}
	}()

	ret := <-rc

	if ret.err != nil {
		logger.Debugf(e, "retrieval error: %s", ret.err)
		return nil, ret.err
	}

	if ret.response == nil {
		logger.Debugf(e, "no response")
		return nil, errors.New("no response")
	}

	defer ret.response.Body.Close()

	if !isContentTypeAllowed(ret.response.Header.Get("Content-Type")) {
		logger.Debugf(e, "disallowed content type %s", ret.response.Header.Get("Content-Type"))
		return nil, DisallowedContentTypeError
	}

	return goquery.NewDocumentFromReader(ret.response.Body)
}

func (r *retriever) RetrieveDocumentSelection(e *irc.Event, params RetrievalParams, selector string) (*goquery.Selection, error) {
	logger := log.Logger()

	var selection result
	res := make(chan result)

	attempts := 0

	for {
		if attempts+1 > params.Retries {
			logger.Debugf(e, "stopping after %d attempts", attempts)
			selection = result{nil, errors.New("reached max attempts")}
			break
		}

		attempts++

		if attempts > 1 {
			delay := (retryDelayOffset * time.Millisecond) + (time.Duration(rand.IntN(params.RetryDelay)) * time.Millisecond)
			logger.Debugf(e, "waiting %s before attempt %d", delay, attempts)
			time.Sleep(delay)
		}

		logger.Debugf(e, "%s %s, attempt %d", params.Method, params.URL, attempts)

		go func() {
			doc, err := r.RetrieveDocument(e, params, DefaultTimeout)
			if err != nil {
				if errors.Is(err, DisallowedContentTypeError) {
					logger.Debugf(e, "exiting due to disallowed content type")
					res <- result{nil, err}
				} else {
					logger.Debugf(e, "retrieval error: %s", err)
				}
				return
			}

			node := doc.Find(selector)
			if node.Nodes == nil {
				logger.Debugf(e, "no nodes found for selector %s", selector)
				return
			}

			res <- result{node.First(), nil}
		}()

		go func() {
			time.Sleep(params.Timeout * time.Millisecond)

			if selection.doc != nil {
				return
			}

			logger.Debugf(e, "retrieval attempt %d timed out", attempts)
			res <- result{nil, RequestTimedOutError}
		}()

		selection = <-res

		if errors.Is(selection.err, RequestTimedOutError) {
			continue
		}

		break
	}

	return selection.doc, selection.err
}

func (r *retriever) Parse(e *irc.Event, doc *goquery.Document, selectors ...string) []*goquery.Selection {
	logger := log.Logger()
	logger.Debugf(e, "parsing document")

	results := make([]*goquery.Selection, 0)

	for _, selector := range selectors {
		nodes := doc.Find(selector)
		if nodes == nil {
			logger.Debugf(e, "no nodes found for selector %s", selector)
			continue
		}

		node := nodes.First()
		if node == nil {
			logger.Debugf(e, "no node found for selector %s", selector)
			continue
		}

		results = append(results, node)
	}

	return results
}

func isContentTypeAllowed(contentType string) bool {
	for _, p := range allowedContentTypePrefixes {
		if strings.HasPrefix(contentType, p) {
			return true
		}
	}
	return false
}

func translateURL(url string) string {
	domain := rootDomain(url)
	switch domain {
	case "x.com":
		return replaceRoot(url, "x.com", "fixupx.com")
	case "twitter.com":
		return replaceRoot(url, "twitter.com", "fxtwitter.com")
	}
	return url
}

func rootDomain(url string) string {
	matches := rootDomainRegexp.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
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
	return rootDomain(url)
}
