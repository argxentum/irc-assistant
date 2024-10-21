package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const sayCommandName = "say"

type sayCommand struct {
	*commandStub
}

func NewSayCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &sayCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (c *sayCommand) Name() string {
	return sayCommandName
}

func (c *sayCommand) Description() string {
	return "Sends a message to the specified channel."
}

func (c *sayCommand) Triggers() []string {
	return []string{"say"}
}

func (c *sayCommand) Usages() []string {
	return []string{"%s <channel> <message>"}
}

func (c *sayCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *sayCommand) CanExecute(e *irc.Event) bool {
	if !c.isCommandEventValid(c, e, 3) {
		return false
	}

	tokens := Tokens(e.Message())
	return irc.IsChannel(tokens[1])
}

func (c *sayCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := tokens[1]
	message := strings.Join(tokens[2:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, message)

	c.SendMessage(e, channel, message)
}
