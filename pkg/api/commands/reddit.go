package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
	"time"
)

type redditCommand struct {
	*commandStub
	subreddit   string
	description string
	triggers    []string
	usages      []string
	retriever   retriever.DocumentRetriever
}

func NewRedditCommand(ctx context.Context, cfg *config.Config, irc irc.IRC, subreddit, description string, triggers, usages []string) Command {
	return &redditCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		subreddit:   subreddit,
		description: description,
		triggers:    triggers,
		usages:      usages,
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *redditCommand) Name() string {
	return fmt.Sprintf("r/%s", c.subreddit)
}

func (c *redditCommand) Description() string {
	return c.description
}

func (c *redditCommand) Triggers() []string {
	return c.triggers
}

func (c *redditCommand) Usages() []string {
	return c.usages
}

func (c *redditCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *redditCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *redditCommand) Execute(e *irc.Event) {
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

const postsTitleMaxLength = 256

func (c *redditCommand) sendPostMessages(e *irc.Event, posts []reddit.Post) {
	content := make([]string, 0)
	for i, post := range posts {
		title := post.Title
		if len(title) == 0 {
			continue
		}
		if len(title) > postsTitleMaxLength {
			title = title[:postsTitleMaxLength] + "..."
		}
		content = append(content, fmt.Sprintf("%s (r/%s, %s)", style.Bold(title), c.subreddit, elapse.TimeDescription(time.Unix(int64(post.Created), 0))))
		content = append(content, post.URL)
		if i < len(posts)-1 {
			content = append(content, " ")
		}
	}

	c.SendMessages(e, e.ReplyTarget(), content)
}
