package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

type RedditCommand struct {
	*commandStub
	subreddit   string
	description string
	triggers    []string
	usages      []string
	retriever   retriever.DocumentRetriever
}

func NewRedditCommand(ctx context.Context, cfg *config.Config, irc irc.IRC, subreddit, description string, triggers, usages []string) Command {
	return &RedditCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		subreddit:   subreddit,
		description: description,
		triggers:    triggers,
		usages:      usages,
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *RedditCommand) Name() string {
	return fmt.Sprintf("r/%s", c.subreddit)
}

func (c *RedditCommand) Description() string {
	return c.description
}

func (c *RedditCommand) Triggers() []string {
	return c.triggers
}

func (c *RedditCommand) Usages() []string {
	return c.usages
}

func (c *RedditCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *RedditCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *RedditCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	query := strings.Join(tokens[1:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), query)

	posts, err := reddit.SearchNewSubredditPosts(c.ctx, c.cfg, c.subreddit, query)
	if err != nil {
		logger.Warningf(e, "unable to retrieve %s posts in r/%s: %s", query, c.subreddit, err)
		c.Replyf(e, "Unable to retrieve r/%s posts", c.subreddit)
		return
	}
	if len(posts) == 0 {
		logger.Warningf(e, "no %s posts in r/%s", query, c.subreddit)
		c.Replyf(e, "No r/%s posts found for %s", c.subreddit, style.Bold(query))
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
		content = append(content, post.Post.URL)

		if post.Comment != nil {
			content = append(content, post.Comment.FormattedBody())
		}

		if bias, ok := repository.GetBiasResult(nil, post.Post.URL, false); ok {
			content = append(content, bias.ShortDescription())
		}

		if i < len(posts)-1 {
			content = append(content, " ")
		}
	}

	c.SendMessages(e, e.ReplyTarget(), content)
}
