package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

func (c *summaryCommand) directRequest(e *irc.Event, url string) (*summary, error) {
	return c.request(e, url, false)
}

func (c *summaryCommand) request(e *irc.Event, url string, impersonated bool) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "request for %s (impersonated: %t)", url, impersonated)

	params := retriever.DefaultParams(url)
	params.Impersonate = impersonated

	doc, err := c.docRetriever.RetrieveDocument(e, params, retriever.DefaultTimeout)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("unable to retrieve %s", url)
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())
	titleAttr, _ := doc.Find("meta[property='og:title']").First().Attr("content")
	titleMeta := strings.TrimSpace(titleAttr)
	descriptionAttr, _ := doc.Find("html meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descriptionAttr)
	h1 := strings.TrimSpace(doc.Find("html body h1").First().Text())

	if len(description) > maximumDescriptionLength {
		description = description[:maximumDescriptionLength] + "..."
	}

	if len(titleAttr) > 0 {
		title = titleMeta
	} else if len(h1) > 0 {
		title = h1
	}

	if isRejectedTitle(title) {
		return nil, rejectedTitleError
	}

	if len(title)+len(description) < minimumTitleLength {
		return nil, summaryTooShortError
	}

	logger.Debugf(e, "title: %s, description: %s", title, description)

	if len(title) > 0 && len(description) > 0 && (len(title)+len(description) < maximumDescriptionLength || len(title) < minimumPreferredTitleLength) {
		if strings.Contains(description, title) || strings.Contains(title, description) {
			if len(description) > len(title) {
				return createSummary(style.Bold(description)), nil
			}
			return createSummary(style.Bold(title)), nil
		}
		return createSummary(fmt.Sprintf("%s: %s", style.Bold(title), description)), nil
	}

	if len(title) > 0 {
		return createSummary(style.Bold(title)), nil
	}

	if len(description) > 0 {
		return createSummary(style.Bold(description)), nil
	}

	return nil, noContentError
}
