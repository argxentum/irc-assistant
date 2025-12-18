package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bobesa/go-domain-util/domainutil"
)

func (c *SummaryCommand) startPageRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()

	searchURL := fmt.Sprintf(startPageSearchURL, url)
	if u, isSlugified := getSearchURLFromSlug(url, startPageSearchURL, false); isSlugified {
		searchURL = u
	}

	logger.Infof(e, "startpage search for %s, search url %s", url, searchURL)

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(searchURL))
	if err != nil {
		logger.Debugf(e, "unable to retrieve startpage search results for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve startpage search results for %s", url)
		return nil, fmt.Errorf("startpage search results doc nil")
	}

	var ch = make(chan pageSearchResult, 1)
	urlDomain := domainutil.Domain(url)

	go func() {
		doc.Root.Find("section#main div.result").EachWithBreak(func(i int, s *goquery.Selection) bool {
			anchorURL := strings.TrimSpace(s.Find("a.result-title").First().AttrOr("href", ""))
			anchorURLDomain := domainutil.Domain(anchorURL)
			if anchorURLDomain != urlDomain {
				logger.Debugf(e, "startpage result anchor domain (%s) does not match url domain (%s), skipping...", anchorURLDomain, urlDomain)
				return true
			}

			title := strings.TrimSpace(s.Find("a.result-title h2").First().Text())
			description := strings.TrimSpace(s.Find("p.description").First().Text())

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
		logger.Debugf(e, "no valid startpage search results for %s (timeout)", url)
		return nil, noContentError
	}

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "startpage title contains url: %s", title)
		return nil, rejectedTitleError
	}

	s, err := c.createSummaryFromTitleAndDescription(title, description)
	if errors.Is(err, rejectedTitleError) {
		logger.Debugf(e, "rejected startpage summary title: %s", title)
		return nil, err
	}
	if errors.Is(err, summaryTooShortError) {
		logger.Debugf(e, "startpage summary too short - title: %s, description: %s", title, description)
		return nil, err
	}
	if errors.Is(err, noContentError) {
		logger.Debugf(e, "startpage summary no content - title: %s, description: %s", title, description)
		return nil, err
	}

	logger.Debugf(e, "startpage search request - title: %s, description: %s", title, description)
	return s, nil
}
