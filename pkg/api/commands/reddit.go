package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const RedditCommandName = "reddit"
const redditPublicURL = "https://reddit.com%s"

type RedditCommand struct {
	*commandStub
}

func NewRedditCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &RedditCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *RedditCommand) Name() string {
	return RedditCommandName
}

func (c *RedditCommand) Description() string {
	return "Searches for a post in the given subreddit on the given topic."
}

func (c *RedditCommand) Triggers() []string {
	return []string{"reddit"}
}

func (c *RedditCommand) Usages() []string {
	return []string{"%s <subreddit> <topic>"}
}

func (c *RedditCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *RedditCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 2)
}

func (c *RedditCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())

	subreddit := tokens[1]
	subreddit = strings.TrimPrefix(subreddit, "/r/")
	subreddit = strings.TrimPrefix(subreddit, "r/")
	query := strings.Join(tokens[2:], " ")

	logger.Infof(e, "⚡ %s [%s/%s] r/%s %s", c.Name(), e.From, e.ReplyTarget(), subreddit, query)

	posts, err := reddit.SearchRelevantSubredditPosts(c.ctx, c.cfg, subreddit, query)
	if err != nil {
		logger.Warningf(e, "unable to retrieve %s posts in r/%s: %s", query, subreddit, err)
		c.Replyf(e, "Unable to retrieve r/%s posts", subreddit)
		return
	}

	if len(posts) == 0 {
		logger.Warningf(e, "no %s posts in r/%s", query, subreddit)
		c.Replyf(e, "No r/%s posts found for %s", subreddit, style.Bold(query))
		return
	}

	c.sendPostMessages(e, posts)
}

func (c *RedditCommand) sendPostMessages(e *irc.Event, posts []reddit.PostWithTopComment) {
	content := make([]string, 0)
	for i, post := range posts {
		title := text.SanitizeSummaryContent(post.Post.Title)
		if len(title) == 0 {
			continue
		}

		content = append(content, post.Post.FormattedTitle())
		content = append(content, fmt.Sprintf(redditPublicURL, post.Post.Permalink))

		if post.Comment != nil {
			content = append(content, post.Comment.FormattedBody())
		}

		source, err := repository.FindSource(post.Post.URL)
		if err != nil {
			log.Logger().Errorf(nil, "error finding source, %s", err)
		}

		if source != nil {
			content = append(content, repository.ShortSourceSummary(source))
		}

		if i < len(posts)-1 {
			content = append(content, " ")
		}
	}

	c.SendMessages(e, e.ReplyTarget(), content)
}
