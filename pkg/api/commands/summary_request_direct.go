package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

func (c *SummaryCommand) directRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "direct request for %s", doc.URL)

	title := strings.TrimSpace(doc.Root.Find("title").First().Text())
	titleAttr, _ := doc.Root.Find("meta[property='og:title']").First().Attr("content")
	titleMeta := strings.TrimSpace(titleAttr)
	descriptionAttr, _ := doc.Root.Find("html meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descriptionAttr)
	h1 := strings.TrimSpace(doc.Root.Find("html body h1").First().Text())

	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	cssIndicators := []string{"{", ":", ";", "}"}
	if text.ContainsAll(title, cssIndicators) {
		title = ""
	}
	if text.ContainsAll(h1, cssIndicators) {
		h1 = ""
	}

	if len(titleAttr) > 0 {
		title = titleMeta
	} else if len(h1) > 0 {
		title = h1
	}

	if c.isRejectedTitle(title) {
		logger.Debugf(e, "rejected direct title: %s", title)
		return nil, rejectedTitleError
	}

	if len(title)+len(description) < minimumTitleLength {
		logger.Debugf(e, "direct summary too short - title: %s, description: %s", title, description)
		return nil, summaryTooShortError
	}

	if len(title) > maximumTitleLength {
		title = title[:maximumTitleLength] + "..."
	}

	logger.Debugf(e, "direct request - title: %s, description: %s", title, description)

	if len(title) > 0 && len(description) > 0 && (len(title)+len(description) < standardMaximumDescriptionLength || len(title) < minimumPreferredTitleLength) {
		if text.MostlyContains(title, description, 0.9) {
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
