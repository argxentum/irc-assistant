package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

func (c *SummaryCommand) startPageRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()
	logger.Infof(e, "trying startpage for %s", url)

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(startPageSearchURL, url)))
	if err != nil {
		logger.Debugf(e, "unable to retrieve startpage search results for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve startpage search results for %s", url)
		return nil, fmt.Errorf("startpage search results doc nil")
	}

	title := strings.TrimSpace(doc.Root.Find("section#main h2").First().Text())

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "startpage title contains url: %s", title)
		return nil, rejectedTitleError
	}

	if len(title) == 0 {
		logger.Debugf(e, "startpage title empty")
		return nil, summaryTooShortError
	}

	if c.isRejectedTitle(title) {
		return nil, fmt.Errorf("rejected title: %s", title)
	}

	return createSummary(style.Bold(title)), nil
}
