package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"strings"
)

func (c *SummaryCommand) duckduckgoRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()

	searchURL := fmt.Sprintf(duckDuckGoSearchURL, url)
	if u, isSlugified := getSearchURLFromSlug(url, duckDuckGoSearchURL, true); isSlugified {
		searchURL = u
	}

	logger.Infof(e, "duckduckgo search for %s, search url %s", url, searchURL)

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(searchURL))
	if err != nil {
		logger.Debugf(e, "unable to retrieve duckduckgo search results for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve duckduckgo search results for %s", url)
		return nil, fmt.Errorf("duckduckgo search results doc nil")
	}

	title := strings.TrimSpace(doc.Root.Find("div.serp__results").First().Find("h2.result__title").First().Text())
	description := strings.TrimSpace(doc.Root.Find("div.serp__results").First().Find("a.result__snippet").First().Text())

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "duckduckgo title contains url: %s", title)
		return nil, rejectedTitleError
	}

	s, err := c.createSummaryFromTitleAndDescription(title, description)
	if errors.Is(err, rejectedTitleError) {
		logger.Debugf(e, "rejected duckduckgo summary title: %s", title)
		return nil, err
	}
	if errors.Is(err, summaryTooShortError) {
		logger.Debugf(e, "duckduckgo summary too short - title: %s, description: %s", title, description)
		return nil, err
	}
	if errors.Is(err, noContentError) {
		logger.Debugf(e, "duckduckgo summary no content - title: %s, description: %s", title, description)
		return nil, err
	}

	logger.Debugf(e, "duckduckgo search request - title: %s, description: %s", title, description)
	return s, nil
}
