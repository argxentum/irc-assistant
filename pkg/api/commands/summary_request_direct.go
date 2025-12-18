package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"errors"
	"strings"
)

func (c *SummaryCommand) directRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "direct request for %s", doc.URL)

	title := strings.TrimSpace(doc.Root.Find("title").First().Text())
	titleAttr, _ := doc.Root.Find("meta[property='og:title']").First().Attr("content")
	titleMeta := strings.TrimSpace(titleAttr)
	descriptionAttr, _ := doc.Root.Find("html meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descriptionAttr)
	h1 := strings.TrimSpace(doc.Root.Find("html body h1").First().Text())

	cssIndicators := []string{"{", ":", ";", "}"}
	if text.ContainsAll(title, cssIndicators) {
		title = ""
	}
	if text.ContainsAll(h1, cssIndicators) {
		h1 = ""
	}

	if len(titleAttr) > 0 {
		title = titleMeta
	} else if len(h1) > 0 {
		title = h1
	}

	s, err := c.createSummaryFromTitleAndDescription(title, description)
	if errors.Is(err, rejectedTitleError) {
		logger.Debugf(e, "rejected direct summary title: %s", title)
		return nil, err
	}
	if errors.Is(err, summaryTooShortError) {
		logger.Debugf(e, "direct summary too short - title: %s, description: %s", title, description)
		return nil, err
	}
	if errors.Is(err, noContentError) {
		logger.Debugf(e, "direct summary no content - title: %s, description: %s", title, description)
		return nil, err
	}

	logger.Debugf(e, "direct request - title: %s, description: %s", title, description)
	return s, nil
}
