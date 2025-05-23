package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const JoinCommandName = "join"

type JoinCommand struct {
	*commandStub
}

func NewJoinCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &JoinCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *JoinCommand) Name() string {
	return JoinCommandName
}

func (c *JoinCommand) Description() string {
	return "Invites to join the specified channel(s)."
}

func (c *JoinCommand) Triggers() []string {
	return []string{"join"}
}

func (c *JoinCommand) Usages() []string {
	return []string{"%s <channel1> [<channel2> ...]"}
}

func (c *JoinCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *JoinCommand) CanExecute(e *irc.Event) bool {
	if !e.IsPrivateMessage() || !c.isCommandEventValid(c, e, 1) {
		return false
	}

	tokens := Tokens(e.Message())
	return irc.IsChannel(tokens[1])
}

func (c *JoinCommand) Execute(e *irc.Event) {
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
