package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"strings"
)

func (c *SummaryCommand) bingRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()

	searchURL := fmt.Sprintf(bingSearchURL, url)
	if u, isSlugified := getSearchURLFromSlugs(url, bingSearchURL); isSlugified {
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

	title := strings.TrimSpace(doc.Root.Find("ol#b_results").First().Find("h2").First().Text())

	if c.isRejectedTitle(title) {
		logger.Debugf(e, "rejected bing title: %s", title)
		return nil, rejectedTitleError
	}

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "bing title contains url': %s", title)
		return nil, rejectedTitleError
	}

	if len(title) == 0 {
		logger.Debugf(e, "bing title empty")
		return nil, summaryTooShortError
	}

	if len(title) < minimumTitleLength {
		logger.Debugf(e, "bing title too short: %s", title)
		return nil, summaryTooShortError
	}

	logger.Debugf(e, "bing request - title: %s", title)

	return createSummary(style.Bold(title)), nil
}
