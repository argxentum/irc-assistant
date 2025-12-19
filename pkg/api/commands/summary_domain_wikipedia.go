package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/api/wikipedia"
	"assistant/pkg/models"
)

func (c *SummaryCommand) parseWikipedia(e *irc.Event, url string) (*summary, *models.Source, error) {
	page, err := wikipedia.GetPageForURL(url, c.cfg.IRC.Nick+" (IRC bot)")
	if err != nil {
		return nil, nil, err
	}

	description := page.Extract
	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	s, err := c.createSummaryFromTitleAndDescription(style.Underline(page.Title), description)
	return s, nil, err
}
