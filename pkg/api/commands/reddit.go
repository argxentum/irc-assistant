package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"strings"
)

const RedditCommandName = "reddit"

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

	task := models.NewProxyRedditSearchRequestTask(e.ReplyTarget(), e.From, subreddit, query, models.RedditSearchSortRelevance)
	if err := queue.GetProxy().Publish(task); err != nil {
		logger.Errorf(e, "error publishing reddit search request, %s", err)
	}
}
