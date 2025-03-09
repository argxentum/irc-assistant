package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const GIFSearchCommandName = "gif_search"

type GIFSearchCommand struct {
	*commandStub
}

func NewGifSearchCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &GIFSearchCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *GIFSearchCommand) Name() string {
	return GIFSearchCommandName
}

func (c *GIFSearchCommand) Description() string {
	return "Searches for a gif on Giphy."
}

func (c *GIFSearchCommand) Triggers() []string {
	return []string{"gif", "gifs", "giphy"}
}

func (c *GIFSearchCommand) Usages() []string {
	return []string{"%s <search>"}
}

func (c *GIFSearchCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *GIFSearchCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *GIFSearchCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], "_") + ".gif"
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), message)
	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s/gifs/%s", c.cfg.Web.ExternalRootURL, message))
}
