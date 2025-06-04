package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"strings"
)

var csf map[string]func(e *irc.Event, doc *retriever.Document) (*summary, error)

func (c *SummaryCommand) contentSummarization() map[string]func(e *irc.Event, doc *retriever.Document) (*summary, error) {
	if csf == nil {
		csf = map[string]func(e *irc.Event, doc *retriever.Document) (*summary, error){
			"https://joinmastodon.org/apps": c.parseMastodon,
		}
	}

	return csf
}

func (c *SummaryCommand) contentSummary(e *irc.Event, doc *retriever.Document) (func(e *irc.Event, doc *retriever.Document) (*summary, error), error) {
	if doc == nil || doc.Body == nil {
		return nil, nil
	}

	payload := string(doc.Body.Data)

	for content, cmd := range c.contentSummarization() {
		if strings.Contains(payload, content) {
			return cmd, nil
		}
	}

	return nil, nil
}
