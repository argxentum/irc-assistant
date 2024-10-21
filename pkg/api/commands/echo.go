package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const echoCommandName = "echo"

type echoCommand struct {
	*commandStub
}

func NewEchoCommand(ctx context.Context, cfg *config.Config, ircSvc irc.IRC) Command {
	return &echoCommand{
		commandStub: newCommandStub(ctx, cfg, ircSvc, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (c *echoCommand) Name() string {
	return echoCommandName
}

func (c *echoCommand) Description() string {
	return "Echoes the given message."
}

func (c *echoCommand) Triggers() []string {
	return []string{"echo"}
}

func (c *echoCommand) Usages() []string {
	return []string{"echo <message>"}
}

func (c *echoCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *echoCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *echoCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], " ")
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), message)
	c.SendMessage(e, e.ReplyTarget(), message)
}
