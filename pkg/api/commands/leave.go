package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const LeaveCommandName = "leave"

type LeaveCommand struct {
	*commandStub
}

func NewLeaveCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &LeaveCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *LeaveCommand) Name() string {
	return LeaveCommandName
}

func (c *LeaveCommand) Description() string {
	return "Leaves the specified channel(s)."
}

func (c *LeaveCommand) Triggers() []string {
	return []string{"leave", "part"}
}

func (c *LeaveCommand) Usages() []string {
	return []string{"%s [<channel1> [<channel2> ...]]"}
}

func (c *LeaveCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *LeaveCommand) CanExecute(e *irc.Event) bool {
	tokens := Tokens(e.Message())
	if e.IsPrivateMessage() {
		return c.isCommandEventValid(c, e, 1) && irc.IsChannel(tokens[1])
	}

	return c.isCommandEventValid(c, e, 0)
}

func (c *LeaveCommand) Execute(e *irc.Event) {
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
