package functions

import (
	"github.com/PuerkitoBio/goquery"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
)

var headerSets = []map[string]string{
	{
		"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language": "en-US,en;q=0.9",
	},
}

var rootDomainRegexp = regexp.MustCompile(`https?://.*?([^.]+\.[a-z]+)(?:/|$)`)

func getDocument(url string, impersonated bool) (*goquery.Document, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if impersonated {
		headers := headerSets[rand.Intn(len(headerSets))]
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, err
	}

	if !isContentTypeAllowed(resp.Header.Get("Content-Type")) {
		return nil, nil
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
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
