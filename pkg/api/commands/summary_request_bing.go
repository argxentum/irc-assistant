package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bobesa/go-domain-util/domainutil"
)

func (c *SummaryCommand) bingRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()

	searchURL := fmt.Sprintf(bingSearchURL, url)
	if u, isSlugified := getSearchURLFromSlug(url, bingSearchURL, false); isSlugified {
		searchURL = u
	}

	logger.Infof(e, "bing search for %s, search url %s", url, searchURL)

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(searchURL))
	if err != nil {
		logger.Debugf(e, "unable to retrieve bing search results for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve bing search results for %s", url)
		return nil, errors.New("bing search results doc nil")
	}

	var ch = make(chan pageSearchResult, 1)
	urlDomain := domainutil.Domain(url)

	go func() {
		doc.Root.Find("ol#b_results li.b_algo").EachWithBreak(func(i int, s *goquery.Selection) bool {
			anchorURL := getBingRedirectURL(strings.TrimSpace(s.Find("h2 a").First().AttrOr("href", "")))
			anchorURLDomain := domainutil.Domain(anchorURL)
			if anchorURLDomain != urlDomain {
				logger.Debugf(e, "bing result anchor domain (%s) does not match url domain (%s), skipping...", anchorURLDomain, urlDomain)
				return true
			}

			title := strings.TrimSpace(s.Find("h2").First().Text())
			description := strings.TrimSpace(s.Find("div.b_caption").First().Text())

			ch <- pageSearchResult{
				title:       title,
				description: description,
			}

			return false
		})
	}()

	var title, description string
	select {
	case res := <-ch:
		title = res.title
		description = res.description
	case <-time.After(3 * time.Second):
		logger.Debugf(e, "no valid bing search results for %s (timeout)", url)
		return nil, noContentError
	}

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "bing title contains url': %s", title)
		return nil, rejectedTitleError
	}

	s, err := c.createSummaryFromTitleAndDescription(title, description)
	if errors.Is(err, rejectedTitleError) {
		logger.Debugf(e, "rejected bing summary title: %s", title)
		return nil, err
	}
	if errors.Is(err, summaryTooShortError) {
		logger.Debugf(e, "bing summary too short - title: %s, description: %s", title, description)
		return nil, err
	}
	if errors.Is(err, noContentError) {
		logger.Debugf(e, "bing summary no content - title: %s, description: %s", title, description)
		return nil, err
	}

	logger.Debugf(e, "bing search request - title: %s, description: %s", title, description)
	return s, nil
}

func getBingRedirectURL(anchorURL string) string {
	if !strings.HasPrefix(anchorURL, "https://www.bing.com/") {
		return anchorURL
	}

	u, err := url.Parse(anchorURL)
	if err != nil {
		return ""
	}
	q := u.Query()

	encoded := q.Get("u")
	if strings.HasPrefix(encoded, "a1") {
		encoded = strings.TrimPrefix(encoded, "a1")
	}

	buf, _ := base64.StdEncoding.DecodeString(encoded)
	if len(buf) == 0 {
		return anchorURL
	}

	return string(buf)
}
