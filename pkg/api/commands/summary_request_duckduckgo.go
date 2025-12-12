package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

func (c *SummaryCommand) duckduckgoRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()

	searchURL := fmt.Sprintf(duckDuckGoSearchURL, url)
	if u, isSlugified := getSearchURLFromSlugs(url, duckDuckGoSearchURL); isSlugified {
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

	title := strings.TrimSpace(doc.Root.Find("div.result__body").First().Find("h2.result__title").First().Text())

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "duckduckgo title contains url: %s", title)
		return nil, rejectedTitleError
	}

	if len(title) == 0 {
		logger.Debugf(e, "duckduckgo title empty")
		return nil, summaryTooShortError
	}

	if c.isRejectedTitle(title) {
		logger.Debugf(e, "rejected duckduckgo title: %s", title)
		return nil, rejectedTitleError
	}

	if len(title) < minimumTitleLength {
		logger.Debugf(e, "duckduckgo title too short: %s", title)
		return nil, summaryTooShortError
	}

	logger.Debugf(e, "duckduckgo request - title: %s", title)

	return createSummary(style.Bold(title)), nil
}
