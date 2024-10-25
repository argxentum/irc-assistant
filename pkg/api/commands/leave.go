package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const leaveCommandName = "leave"

type leaveCommand struct {
	*commandStub
}

func NewLeaveCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &leaveCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (c *leaveCommand) Name() string {
	return leaveCommandName
}

func (c *leaveCommand) Description() string {
	return "Leaves the specified channel(s)."
}

func (c *leaveCommand) Triggers() []string {
	return []string{"leave", "part"}
}

func (c *leaveCommand) Usages() []string {
	return []string{"%s [<channel1> [<channel2> ...]]"}
}

func (c *leaveCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *leaveCommand) CanExecute(e *irc.Event) bool {
	tokens := Tokens(e.Message())
	if e.IsPrivateMessage() {
		return c.isCommandEventValid(c, e, 1) && irc.IsChannel(tokens[1])
	}

	return c.isCommandEventValid(c, e, 0)
}

func (c *leaveCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channels := tokens[1:]

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), strings.Join(channels, ", "))

	if len(tokens) == 1 && !e.IsPrivateMessage() {
		c.irc.Part(e.ReplyTarget())
		logger.Infof(e, "left %s", e.ReplyTarget())
		return
	}

	for _, channel := range channels {
		if !irc.IsChannel(channel) {
			continue
		}

		c.irc.Part(channel)
		logger.Infof(e, "left %s", channel)
	}
}