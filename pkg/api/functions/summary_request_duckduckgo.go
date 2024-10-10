package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

func (f *summaryFunction) duckduckgoRequest(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "trying duckduckgo for %s", url)

	doc, err := f.docRetriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(duckDuckGoSearchURL, url)), retriever.DefaultTimeout)
	if err != nil {
		logger.Debugf(e, "unable to retrieve duckduckgo search results for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve duckduckgo search results for %s", url)
		return nil, fmt.Errorf("duckduckgo search results doc nil")
	}

	title := strings.TrimSpace(doc.Find("div.result__body").First().Find("h2.result__title").First().Text())

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "duckduckgo title contains url: %s", title)
		return nil, rejectedTitleError
	}

	if len(title) == 0 {
		logger.Debugf(e, "duckduckgo title empty")
		return nil, summaryTooShortError
	}

	return createSummary(style.Bold(title)), nil
}
