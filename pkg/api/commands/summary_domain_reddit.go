package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/summary"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
)

func (c *SummaryCommand) parseReddit(e *irc.Event, url string) (*summaryResult, *models.Source, error) {
	logger := log.Logger()
	logger.Debugf(e, "Reddit summary path: trying oEmbed for %s", url)

	result, metadata, err := c.oEmbedSummary(e, redditOEmbedURL, url)
	if err != nil {
		logger.Debugf(e, "Reddit summary path: oEmbed failed for %s: %v; falling back to residential proxy", url, err)
		return nil, nil, fmt.Errorf("reddit oEmbed failed for %s: %w", url, err)
	}
	if result == nil || metadata == nil {
		logger.Debugf(e, "Reddit summary path: oEmbed returned no usable summary for %s; falling back to residential proxy", url)
		return nil, nil, fmt.Errorf("reddit oEmbed returned no usable summary for %s", url)
	}

	title := summary.Sanitize(metadata.Title)
	if c.isRejectedTitle(title) {
		logger.Infof(e, "rejected reddit domain title: %s", title)
		return nil, nil, rejectedTitleError
	}
	logger.Debugf(e, "Reddit summary path: oEmbed succeeded for %s", url)

	source, err := repository.FindSource(url)
	if err != nil {
		logger.Errorf(e, "Reddit oEmbed source check failed for %s: %v", url, err)
	} else if source != nil {
		logger.Debugf(e, "Reddit oEmbed source check: matched source %s", source.ID)
	} else {
		logger.Debugf(e, "Reddit oEmbed source check: no configured source matched")
	}

	return result, source, nil
}
