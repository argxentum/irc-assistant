package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"fmt"
	"strings"
)

func (c *summaryCommand) parseMastodon(e *irc.Event, url string) (*summary, error) {
	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(url), 500)
	if err != nil {
		return nil, err
	}

	titleAttr, _ := doc.Find("meta[property='og:title']").First().Attr("content")
	title := strings.TrimSpace(titleAttr)
	descriptionAttr, _ := doc.Find("html meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descriptionAttr)

	if len(description) > maximumDescriptionLength {
		description = description[:maximumDescriptionLength] + "..."
	}

	if len(title) == 0 {
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}

	if isRejectedTitle(title) {
		return nil, fmt.Errorf("rejected title: %s", title)
	}

	if len(title)+len(description) < minimumTitleLength {
		return nil, fmt.Errorf("title and description too short, title: %s, description: %s", title, description)
	}

	if len(description) > 0 {
		return createSummary(fmt.Sprintf("%s • %s", style.Bold(description), title)), nil
	} else {
		return createSummary(title), nil
	}
}
