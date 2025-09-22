package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"fmt"
	"strings"

	"github.com/bobesa/go-domain-util/domainutil"
)

func (c *SummaryCommand) braveSearchRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()
	logger.Infof(e, "trying brave search for %s", url)

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(braveSearchURL, url)))
	if err != nil {
		logger.Debugf(e, "unable to retrieve brave search search results for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve brave search search results for %s", url)
		return nil, fmt.Errorf("brave search search results doc nil")
	}

	result := doc.Root.Find("div#results div.snippet")
	a := strings.TrimSpace(result.Find("a").First().AttrOr("href", ""))
	aDomain := domainutil.Domain(a)
	uDomain := domainutil.Domain(url)
	if aDomain != uDomain {
		logger.Debugf(e, "brave search anchor domain (%s) does not match url domain (%s)", aDomain, uDomain)
		return nil, fmt.Errorf("invalid search result")
	}

	title := strings.TrimSpace(result.Find("div.title").First().Text())
	description := strings.TrimSpace(result.Find("div.snippet-description").First().Text())

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "brave search title contains url: %s", title)
		return nil, rejectedTitleError
	}

	if len(title) == 0 {
		logger.Debugf(e, "brave search title empty")
		return nil, summaryTooShortError
	}

	if c.isRejectedTitle(title) {
		logger.Debugf(e, "rejected brave search title: %s", title)
		return nil, rejectedTitleError
	}

	if len(title) < minimumTitleLength {
		logger.Debugf(e, "brave search title too short: %s", title)
		return nil, summaryTooShortError
	}

	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	logger.Debugf(e, "brave search request - title: %s, description: %s", title, description)

	if len(description) == 0 {
		return createSummary(style.Bold(title)), nil
	}

	return createSummary(style.Bold(title) + ": " + description), nil
}
