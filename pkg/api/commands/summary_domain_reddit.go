package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"assistant/pkg/models"
)

func (c *SummaryCommand) parseReddit(e *irc.Event, url string) (*summary, *models.Source, error) {
	logger := log.Logger()

	messages, err := reddit.Summarize(c.ctx, c.cfg, url)
	if err != nil {
		return nil, nil, err
	}

	if len(messages) == 0 {
		return nil, nil, nil
	}

	title := text.SanitizeSummaryContent(messages[0])
	if c.isRejectedTitle(title) {
		logger.Infof(e, "rejected reddit domain title: %s", title)
		return nil, nil, rejectedTitleError
	}

	s := createSummary(messages...)

	source, err := repository.FindSource(url)
	if err != nil {
		logger.Errorf(nil, "error finding source, %s", err)
	}

	return s, source, nil
}
