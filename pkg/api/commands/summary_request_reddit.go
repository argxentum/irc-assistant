package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"time"
)

func (c *summaryCommand) redditRequest(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "reddit request for %s", url)

	posts, err := reddit.SearchPostsForURL(c.ctx, c.cfg, url)
	if err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return nil, errors.New("no posts found")
	}

	title := text.Sanitize(posts[0].Post.Title)
	if len(title) == 0 {
		return nil, nil
	}

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s (r/%s, %s)", style.Bold(title), posts[0].Post.Subreddit, elapse.TimeDescription(time.Unix(int64(posts[0].Post.Created), 0))))

	if posts[0].Comment != nil {
		comment := text.Sanitize(posts[0].Comment.Body)
		messages = append(messages, fmt.Sprintf("Top comment (by u/%s): %s", posts[0].Comment.Author, style.Italics(comment)))
	}

	return createSummary(messages...), nil
}
