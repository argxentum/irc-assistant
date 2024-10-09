package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"strings"
)

var csf map[string]func(e *irc.Event, url string) (*summary, error)

func (f *summaryFunction) contentSummarization() map[string]func(e *irc.Event, url string) (*summary, error) {
	if csf == nil {
		csf = map[string]func(e *irc.Event, url string) (*summary, error){
			"https://joinmastodon.org/apps": f.parseMastodon,
		}
	}

	return csf
}

func (f *summaryFunction) contentSummary(e *irc.Event, url string) (func(e *irc.Event, url string) (*summary, error), error) {
	if len(url) == 0 {
		return nil, nil
	}

	body, err := f.bodyRetriever.RetrieveBody(e, retriever.DefaultParams(url), 500)
	if err != nil {
		return nil, err
	}

	if body == nil {
		return nil, nil
	}

	payload := string(body)

	for content, fn := range f.contentSummarization() {
		if strings.Contains(payload, content) {
			return fn, nil
		}
	}

	return nil, nil
}
