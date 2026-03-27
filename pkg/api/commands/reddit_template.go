package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
	"strings"
)

type RedditTemplateCommand struct {
	*commandStub
	subreddit   string
	description string
	triggers    []string
	usages      []string
}

func NewRedditTemplateCommand(ctx context.Context, cfg *config.Config, irc irc.IRC, subreddit, description string, triggers, usages []string) Command {
	return &RedditTemplateCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		subreddit:   subreddit,
		description: description,
		triggers:    triggers,
		usages:      usages,
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

	task := models.NewProxyRedditSearchRequestTask(e.ReplyTarget(), e.From, c.subreddit, query, models.RedditSearchSortNew)
	if err := queue.GetProxy().Publish(task); err != nil {
		logger.Errorf(e, "error publishing reddit search request, %s", err)
	}
}
