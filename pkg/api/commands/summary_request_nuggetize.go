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

func (c *SummaryCommand) nuggetizeRequest(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "nuggetize request for %s", url)

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf("https://nug.zip/%s", url)))
	if err != nil {
		logger.Debugf(e, "unable to retrieve nuggetize summary for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve nuggetize summary for %s", url)
		return nil, errors.New("nuggetize summary doc nil")
	}

	title := strings.TrimSpace(doc.Root.Find("span.title").First().Text())

	if c.isRejectedTitle(title) {
		logger.Debugf(e, "rejected title: %s", title)
		return nil, rejectedTitleError
	}

	if len(title) < minimumTitleLength {
		logger.Debugf(e, "nuggetize title too short: %s", title)
		return nil, summaryTooShortError
	}

	return createSummary(style.Bold(title)), nil
}
