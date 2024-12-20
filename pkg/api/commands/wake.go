package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const WakeCommandName = "wake"

type WakeCommand struct {
	*commandStub
}

func NewWakeCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &WakeCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (c *WakeCommand) Name() string {
	return WakeCommandName
}

func (c *WakeCommand) Description() string {
	return "Wakes the bot, enabling it across all channels."
}

func (c *WakeCommand) Triggers() []string {
	return []string{"wake"}
}

func (c *WakeCommand) Usages() []string {
	return []string{"%s"}
}

func (c *WakeCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *WakeCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *WakeCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", c.Name(), e.From, e.ReplyTarget())

	if c.ctx.Session().IsAwake {
		logger.Warningf(e, "already awake")
		c.Replyf(e, "Already awake.")
		return
	}

	c.ctx.Session().IsAwake = true
	logger.Debug(e, "awake")
	c.SendMessage(e, e.ReplyTarget(), "Now awake.")
}
