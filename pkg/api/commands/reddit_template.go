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

type RedditTemplateCommand struct {
	*commandStub
	subreddit   string
	description string
	triggers    []string
	usages      []string
	retriever   retriever.DocumentRetriever
}

func NewRedditTemplateCommand(ctx context.Context, cfg *config.Config, irc irc.IRC, subreddit, description string, triggers, usages []string) Command {
	return &RedditTemplateCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		subreddit:   subreddit,
		description: description,
		triggers:    triggers,
		usages:      usages,
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *RedditTemplateCommand) Name() string {
	return fmt.Sprintf("r/%s", c.subreddit)
}

func (c *RedditTemplateCommand) Description() string {
	return c.description
}

func (c *RedditTemplateCommand) Triggers() []string {
	return c.triggers
}

func (c *RedditTemplateCommand) Usages() []string {
	return c.usages
}

func (c *RedditTemplateCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *RedditTemplateCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *RedditTemplateCommand) Execute(e *irc.Event) {
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

func (c *RedditTemplateCommand) sendPostMessages(e *irc.Event, posts []reddit.PostWithTopComment) {
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
