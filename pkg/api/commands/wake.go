package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const wakeCommandName = "wake"

type wakeCommand struct {
	*commandStub
}

func NewWakeCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &wakeCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (c *wakeCommand) Name() string {
	return wakeCommandName
}

func (c *wakeCommand) Description() string {
	return "Wakes the bot, enabling it across all channels."
}

func (c *wakeCommand) Triggers() []string {
	return []string{"wake"}
}

func (c *wakeCommand) Usages() []string {
	return []string{"%s"}
}

func (c *wakeCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *wakeCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *wakeCommand) Execute(e *irc.Event) {
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
