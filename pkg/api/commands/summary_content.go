package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"strings"
)

var csf map[string]func(e *irc.Event, url string) (*summary, error)

func (c *SummaryCommand) contentSummarization() map[string]func(e *irc.Event, url string) (*summary, error) {
	if csf == nil {
		csf = map[string]func(e *irc.Event, url string) (*summary, error){
			"https://joinmastodon.org/apps": c.parseMastodon,
		}
	}

	return csf
}

func (c *SummaryCommand) contentSummary(e *irc.Event, url string) (func(e *irc.Event, url string) (*summary, error), error) {
	if len(url) == 0 {
		return nil, nil
	}

	body, err := c.bodyRetriever.RetrieveBody(e, retriever.DefaultParams(url).WithTimeout(500))
	if err != nil {
		return nil, err
	}

	if body == nil {
		return nil, nil
	}

	payload := string(body.Data)

	for content, cmd := range c.contentSummarization() {
		if strings.Contains(payload, content) {
			return cmd, nil
		}
	}

	return nil, nil
}
