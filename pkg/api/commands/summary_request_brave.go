package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"strings"

	"github.com/bobesa/go-domain-util/domainutil"
)

func (c *SummaryCommand) braveSearchRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()

	searchURL := fmt.Sprintf(braveSearchURL, url)
	if u, isSlugified := getSearchURLFromSlug(url, braveSearchURL, true); isSlugified {
		searchURL = u
	}

	logger.Infof(e, "brave search for %s, search url %s", url, searchURL)

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(searchURL))
	if err != nil {
		logger.Debugf(e, "unable to retrieve brave search search results for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve brave search search results for %s", url)
		return nil, fmt.Errorf("brave search search results doc nil")
	}

	result := doc.Root.Find("div#results div.snippet")
	a := strings.TrimSpace(result.Find("a").First().AttrOr("href", ""))
	aDomain := domainutil.Domain(a)
	uDomain := domainutil.Domain(url)
	if aDomain != uDomain {
		logger.Debugf(e, "brave search anchor domain (%s) does not match url domain (%s)", aDomain, uDomain)
		return nil, fmt.Errorf("invalid search result")
	}

	title := strings.TrimSpace(result.Find("div.title").First().Text())
	desc1 := strings.TrimSpace(result.Find("div.snippet-description").First().Text())
	desc2 := strings.TrimSpace(result.Find("div.generic-snippet div.content").First().Text())

	description := desc1
	if description == "" {
		description = desc2
	}

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "brave search title contains url: %s", title)
		return nil, rejectedTitleError
	}

	s, err := c.createSummaryFromTitleAndDescription(title, description)
	if errors.Is(err, rejectedTitleError) {
		logger.Debugf(e, "rejected brave summary title: %s", title)
		return nil, err
	}
	if errors.Is(err, summaryTooShortError) {
		logger.Debugf(e, "brave summary too short - title: %s, description: %s", title, description)
		return nil, err
	}
	if errors.Is(err, noContentError) {
		logger.Debugf(e, "brave summary no content - title: %s, description: %s", title, description)
		return nil, err
	}

	logger.Debugf(e, "brave search request - title: %s, description: %s", title, description)
	return s, nil
}
