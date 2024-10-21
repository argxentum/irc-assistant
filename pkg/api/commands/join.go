package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const joinCommandName = "join"

type joinCommand struct {
	*commandStub
}

func NewJoinCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &joinCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (c *joinCommand) Name() string {
	return joinCommandName
}

func (c *joinCommand) Description() string {
	return "Invites to join the specified channel(s)."
}

func (c *joinCommand) Triggers() []string {
	return []string{"join"}
}

func (c *joinCommand) Usages() []string {
	return []string{"%s <channel1> [<channel2> ...]"}
}

func (c *joinCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *joinCommand) CanExecute(e *irc.Event) bool {
	if !e.IsPrivateMessage() || !c.isCommandEventValid(c, e, 1) {
		return false
	}

	tokens := Tokens(e.Message())
	return irc.IsChannel(tokens[1])
}

func (c *joinCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channels := tokens[1:]

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), strings.Join(channels, ", "))

	for _, channel := range channels {
		if !irc.IsChannel(channel) {
			continue
		}

		c.irc.Join(tokens[1])
		logger.Infof(e, "joined %s", channel)
	}
}
