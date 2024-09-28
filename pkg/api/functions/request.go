package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"math/rand"
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

const requestTimeoutMillis = 1000

var disallowedContentTypeError = errors.New("disallowed content type")

var rootDomainRegexp = regexp.MustCompile(`https?://.*?([^.]+\.[a-z]+)(?:/|$)`)

type docResult struct {
	doc *goquery.Document
	err error
}

func (f *FunctionStub) getDocument(e *irc.Event, url string, impersonated bool) (*goquery.Document, error) {
	logger := log.Logger()
	logger.Debugf(e, "GET %s", url)

	res := make(chan docResult)
	go func() {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			logger.Debugf(e, "error getting url, %s", err)
			res <- docResult{nil, err}
		}

		if impersonated {
			headers := headerSets[rand.Intn(len(headerSets))]
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			if err != nil {
				logger.Debugf(e, "error getting url (status %d), %s", resp.StatusCode, err)
			} else {
				logger.Debugf(e, "error getting url (status %s)", resp.Status)
			}
			res <- docResult{nil, err}
		}

		if !isContentTypeAllowed(resp.Header.Get("Content-Type")) {
			res <- docResult{nil, disallowedContentTypeError}
		}

		defer resp.Body.Close()

		logger.Debugf(e, "start parsing document")
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		logger.Debugf(e, "finished parsing document")
		if err != nil {
			logger.Debugf(e, "error parsing document, %s", err)
			res <- docResult{nil, err}
		}

		res <- docResult{doc, nil}
	}()

	go func() {
		time.Sleep(requestTimeoutMillis * time.Millisecond)
		res <- docResult{nil, errors.New("request timed out")}
	}()

	docRes := <-res
	return docRes.doc, docRes.err
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
