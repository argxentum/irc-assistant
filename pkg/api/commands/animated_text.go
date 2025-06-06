package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const AnimatedTextCommandName = "animated_text"

type AnimatedTextCommand struct {
	*commandStub
}

func NewAnimatedTextCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &AnimatedTextCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *AnimatedTextCommand) Name() string {
	return AnimatedTextCommandName
}

func (c *AnimatedTextCommand) Description() string {
	return "Displays the given text as an animation."
}

func (c *AnimatedTextCommand) Triggers() []string {
	return []string{"animate", "animated", "text"}
}

func (c *AnimatedTextCommand) Usages() []string {
	return []string{"%s <text>"}
}

func (c *AnimatedTextCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *AnimatedTextCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *AnimatedTextCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], "_") + ".gif"
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), message)
	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s/animated/%s", c.cfg.Web.ExternalRootURL, message))
}
