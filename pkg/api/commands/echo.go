package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const EchoCommandName = "echo"

type EchoCommand struct {
	*commandStub
}

func NewEchoCommand(ctx context.Context, cfg *config.Config, ircSvc irc.IRC) Command {
	return &EchoCommand{
		commandStub: newCommandStub(ctx, cfg, ircSvc, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *EchoCommand) Name() string {
	return EchoCommandName
}

func (c *EchoCommand) Description() string {
	return "Echoes the given message."
}

func (c *EchoCommand) Triggers() []string {
	return []string{"echo"}
}

func (c *EchoCommand) Usages() []string {
	return []string{"%s <message>"}
}

func (c *EchoCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *EchoCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *EchoCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], " ")
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), message)
	c.SendMessage(e, e.ReplyTarget(), message)
}
