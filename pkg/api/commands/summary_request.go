package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"errors"
)

var rejectedTitleError = errors.New("rejected title")
var summaryTooShortError = errors.New("summary too short")
var noContentError = errors.New("no summary content")

var rsf []func(e *irc.Event, doc *retriever.Document) (*summary, error)

func (c *SummaryCommand) requestChain() []func(e *irc.Event, doc *retriever.Document) (*summary, error) {
	if rsf == nil {
		rsf = []func(e *irc.Event, doc *retriever.Document) (*summary, error){
			c.redditRequest,
			c.directRequest,
			c.firecrawlRequest,
			c.startPageRequest,
			c.duckduckgoRequest,
			c.bingRequest,
			c.nuggetizeRequest,
		}
	}

	return rsf
}

func (c *SummaryCommand) summarize(e *irc.Event, doc *retriever.Document) (*summary, error) {
	logger := log.Logger()

	for _, cmd := range c.requestChain() {
		s, err := cmd(e, doc)
		if err != nil {
			continue
		}
		if s == nil {
			continue
		}

		return s, nil
	}

	logger.Debugf(e, "no request summary found")
	return nil, nil
}
