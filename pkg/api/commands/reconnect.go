package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"time"
)

const ReconnectCommandName = "reconnect"

type ReconnectCommand struct {
	*commandStub
}

func NewReconnectCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &ReconnectCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *ReconnectCommand) Name() string {
	return ReconnectCommandName
}

func (c *ReconnectCommand) Description() string {
	return "Disconnects and reconnects after the specified interval, or as soon as possible if none is given."
}

func (c *ReconnectCommand) Triggers() []string {
	return []string{"reconnect", "restart"}
}

func (c *ReconnectCommand) Usages() []string {
	return []string{"%s [<interval>]"}
}

func (c *ReconnectCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *ReconnectCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *ReconnectCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	interval := "0s"
	if len(tokens) >= 2 {
		interval = tokens[1]
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), interval)

	seconds, err := elapse.ParseDuration(interval)
	if err != nil {
		logger.Errorf(e, "error parsing interval, %s", err)
		c.Replyf(e, "invalid interval, see %s for help", style.Bold(fmt.Sprintf("%s%s", c.cfg.Commands.Prefix, registry.Command(ReconnectCommandName).Triggers()[0])))
		return
	}

	task := models.NewReconnectTask(time.Now().Add(seconds))
	err = firestore.Get().AddTask(task)
	if err != nil {
		logger.Errorf(e, "error adding task, %s", err)
		return
	}

	c.Replyf(e, "disconnecting now, reconnecting %s", style.Bold(elapse.TimeDescription(task.DueAt)))

	c.irc.Disconnect()
}
