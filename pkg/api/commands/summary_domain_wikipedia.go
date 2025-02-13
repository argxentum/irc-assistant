package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/api/wikipedia"
	"fmt"
)

func (c *SummaryCommand) parseWikipedia(e *irc.Event, url string) (*summary, error) {
	page, err := wikipedia.GetPageForURL(url)
	if err != nil {
		return nil, err
	}

	description := page.Summary
	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	return createSummary(fmt.Sprintf("%s • %s", style.Underline(style.Bold(page.Title)), description)), nil
}
