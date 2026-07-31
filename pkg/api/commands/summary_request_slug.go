package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
)

func (c *SummaryCommand) slugSearchRequest(e *irc.Event, doc *retriever.Document) (*summaryResult, error) {
	if _, isSlugified := getSearchURLFromSlug(doc.URL, braveSearchURL, true); !isSlugified {
		return nil, nil
	}

	log.Logger().Debugf(e, "trying slug search fallback for %s", doc.URL)
	requests := []func(*irc.Event, *retriever.Document) (*summaryResult, error){
		c.braveSlugSearchRequest,
		c.startPageSlugSearchRequest,
		c.duckduckgoSlugSearchRequest,
		c.bingSlugSearchRequest,
	}
	for _, request := range requests {
		result, err := request(e, doc)
		if err != nil || result == nil {
			continue
		}
		return result, nil
	}

	return nil, nil
}
