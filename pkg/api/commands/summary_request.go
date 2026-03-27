package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"errors"
	"sync"
)

var rejectedTitleError = errors.New("rejected title")
var summaryTooShortError = errors.New("summary too short")
var noContentError = errors.New("no summary content")

var rsfOnce sync.Once
var rsf []func(e *irc.Event, doc *retriever.Document) (*summaryResult, error)

func (c *SummaryCommand) requestChain() []func(e *irc.Event, doc *retriever.Document) (*summaryResult, error) {
	rsfOnce.Do(func() {
		rsf = []func(e *irc.Event, doc *retriever.Document) (*summaryResult, error){
			//c.redditRequest,
			c.directRequest,
			c.proxySummaryRequest,
			c.braveSearchRequest,
			//c.firecrawlRequest,
			c.startPageRequest,
			c.duckduckgoRequest,
			c.bingRequest,
			//c.nuggetizeRequest,
		}
	})

	return rsf
}

func (c *SummaryCommand) summarize(e *irc.Event, doc *retriever.Document) (*summaryResult, error) {
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
