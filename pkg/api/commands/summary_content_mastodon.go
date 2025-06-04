package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"fmt"
	"strings"
	"time"
)

func (c *SummaryCommand) parseMastodon(e *irc.Event, url string) (*summary, error) {
	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(url))
	if err != nil {
		return nil, err
	}

	titleAttr, _ := doc.Root.Find("meta[property='og:title']").First().Attr("content")
	title := strings.TrimSpace(titleAttr)
	descriptionAttr, _ := doc.Root.Find("html meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descriptionAttr)
	published := doc.Root.Find("meta[property='og:published_time']").First().AttrOr("content", "")

	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	if len(title) == 0 {
		title = strings.TrimSpace(doc.Root.Find("title").First().Text())
	}

	if c.isRejectedTitle(title) {
		return nil, fmt.Errorf("rejected title: %s", title)
	}

	if len(title)+len(description) < minimumTitleLength {
		return nil, fmt.Errorf("title and description too short, title: %s, description: %s", title, description)
	}

	var publishedTime time.Time
	if len(published) > 0 {
		publishedTime, err = time.Parse(time.RFC3339, published)
	}

	if len(description) > 0 && !publishedTime.IsZero() {
		return createSummary(fmt.Sprintf("%s • %s • %s", style.Bold(description), title, elapse.TimeDescription(publishedTime))), nil
	} else if len(description) > 0 {
		return createSummary(fmt.Sprintf("%s • %s", style.Bold(description), title)), nil
	} else {
		return createSummary(title), nil
	}
}
