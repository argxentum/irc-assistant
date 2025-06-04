package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"errors"
)

func (c *SummaryCommand) redditRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()
	logger.Infof(e, "reddit request for %s", url)

	posts, err := reddit.SearchPostsForURL(c.ctx, c.cfg, url)
	if err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return nil, errors.New("no posts found")
	}

	title := text.SanitizeSummaryContent(posts[0].Post.Title)
	if len(title) == 0 {
		return nil, nil
	}

	messages := make([]string, 0)
	messages = append(messages, posts[0].Post.FormattedTitle())

	if posts[0].Comment != nil {
		messages = append(messages, posts[0].Comment.FormattedBody())
	}

	return createSummary(messages...), nil
}
