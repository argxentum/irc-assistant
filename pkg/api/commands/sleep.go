package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
)

const sleepCommandName = "sleep"

type sleepCommand struct {
	*commandStub
}

func NewSleepCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &sleepCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (c *sleepCommand) Name() string {
	return sleepCommandName
}

func (c *sleepCommand) Description() string {
	return "Puts the bot to sleep, disabling it across all channels until awakened."
}

func (c *sleepCommand) Triggers() []string {
	return []string{"sleep"}
}

func (c *sleepCommand) Usages() []string {
	return []string{"%s"}
}

func (c *sleepCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *sleepCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *sleepCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", c.Name(), e.From, e.ReplyTarget())

	wakeTrigger := ""
	for k, v := range registry.Commands() {
		if k == wakeCommandName {
			if len(v.Triggers()) > 0 {
				wakeTrigger = fmt.Sprintf("%s%s", c.cfg.Commands.Prefix, v.Triggers()[0])
			}
		}
	}

	c.ctx.Session().IsAwake = false
	logger.Debug(e, "sleeping")
	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("Sleeping until awoken with %s.", style.Italics(wakeTrigger)))
}
