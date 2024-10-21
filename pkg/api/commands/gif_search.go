package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const gifSearchCommandName = "gifSearch"

type gifSearchCommand struct {
	*commandStub
}

func NewGifSearchCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &gifSearchCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *gifSearchCommand) Name() string {
	return gifSearchCommandName
}

func (c *gifSearchCommand) Description() string {
	return "Searches for a gif on Giphy."
}

func (c *gifSearchCommand) Triggers() []string {
	return []string{"gif", "gifs", "giphy"}
}

func (c *gifSearchCommand) Usages() []string {
	return []string{"%s <search>"}
}

func (c *gifSearchCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *gifSearchCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *gifSearchCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], "_") + ".gif"
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), message)
	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s/gifs/%s", c.cfg.Web.ExternalRootURL, message))
}
