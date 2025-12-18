package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"strings"
)

func (c *SummaryCommand) nuggetizeRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
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

	s, err := c.createSummaryFromTitleAndDescription(title, "")
	if errors.Is(err, rejectedTitleError) {
		logger.Debugf(e, "rejected nuggetize summary title: %s", title)
		return nil, err
	}
	if errors.Is(err, summaryTooShortError) {
		logger.Debugf(e, "nuggetize summary too short - title: %s", title)
		return nil, err
	}
	if errors.Is(err, noContentError) {
		logger.Debugf(e, "nuggetize summary no content - title: %s", title)
		return nil, err
	}

	logger.Debugf(e, "nuggetize search request - title: %s", title)
	return s, nil
}
