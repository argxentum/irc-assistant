package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"errors"
)

var rejectedTitleError = errors.New("rejected title")
var summaryTooShortError = errors.New("summary too short")
var noContentError = errors.New("no summary content")

var rsf []func(e *irc.Event, url string) (*summary, error)

func (c *summaryCommand) requestChain() []func(e *irc.Event, url string) (*summary, error) {
	if rsf == nil {
		rsf = []func(e *irc.Event, url string) (*summary, error){
			c.redditRequest,
			c.directRequest,
			c.impersonatedRequest,
			c.nuggetizeRequest,
			c.duckduckgoRequest,
			c.bingRequest,
			c.startPageRequest,
		}
	}

	return rsf
}

func (c *summaryCommand) summarize(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()

	for _, cmd := range c.requestChain() {
		s, err := cmd(e, url)
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
