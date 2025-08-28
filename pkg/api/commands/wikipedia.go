package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/api/wikipedia"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"net/url"
	"strings"
)

const WikipediaCommandName = "wiki"

type WikipediaCommand struct {
	*commandStub
}

func NewWikipediaCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &WikipediaCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *WikipediaCommand) Name() string {
	return WikipediaCommandName
}

func (c *WikipediaCommand) Description() string {
	return "Searches wikipedia for the specified query."
}

func (c *WikipediaCommand) Triggers() []string {
	return []string{"wiki"}
}

func (c *WikipediaCommand) Usages() []string {
	return []string{"%s <query>"}
}

func (c *WikipediaCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *WikipediaCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *WikipediaCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	query := strings.Join(tokens[1:], " ")
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), query)

	page, err := wikipedia.Search(query, c.cfg.IRC.Nick+" (IRC bot)")
	if err != nil {
		c.Replyf(e, "Unable to search Wikipedia for %s", style.Bold(query))
		return
	}

	if page == nil {
		c.Replyf(e, "No results found for %s", style.Bold(query))
		return
	}

	description := page.Extract
	if len(description) > extendedMaximumDescriptionLength {
		description = description[:extendedMaximumDescriptionLength] + "..."
	}

	messages := []string{
		fmt.Sprintf("%s: %s", style.Bold(style.Underline(page.Title)), description),
	}

	if u, err := url.QueryUnescape(page.ContentURLs.Desktop.Page); err == nil {
		messages = append(messages, u)
	}

	c.SendMessages(e, e.ReplyTarget(), messages)
}
