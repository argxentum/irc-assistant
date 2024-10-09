package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"strings"
)

func (f *summaryFunction) bingRequest(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "bing request for %s", url)

	doc, err := f.docRetriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(bingSearchURL, url)), retriever.DefaultTimeout)
	if err != nil {
		logger.Debugf(e, "unable to retrieve bing search results for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve bing search results for %s", url)
		return nil, errors.New("bing search results doc nil")
	}

	title := strings.TrimSpace(doc.Find("ol#b_results").First().Find("h2").First().Text())

	if isRejectedTitle(title) {
		logger.Debugf(e, "rejected title: %s", title)
		return nil, rejectedTitleError
	}

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "bing title contains url': %s", title)
		return nil, rejectedTitleError
	}

	if len(title) == 0 {
		logger.Debugf(e, "bing title empty")
		return nil, summaryTooShortError
	}

	return &summary{style.Bold(title)}, nil
}
