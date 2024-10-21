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

const tempBanCommandName = "tempban"

type tempBanCommand struct {
	*commandStub
}

func NewTempBanCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &tempBanCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *tempBanCommand) Name() string {
	return tempBanCommandName
}

func (c *tempBanCommand) Description() string {
	return "Temporarily bans the specified user from the channel for the specified duration."
}

func (c *tempBanCommand) Triggers() []string {
	return []string{"tempban", "tb"}
}

func (c *tempBanCommand) Usages() []string {
	return []string{"%s <duration> <mask>"}
}

func (c *tempBanCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *tempBanCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 2)
}

func (c *tempBanCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()

	duration := tokens[1]
	mask := tokens[2]

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, mask)

	seconds, err := elapse.ParseDuration(duration)
	if err != nil {
		logger.Errorf(e, "error parsing duration, %s", err)
		c.Replyf(e, "invalid duration, see %s for help", style.Bold(fmt.Sprintf("%s%s", c.cfg.Commands.Prefix, registry.Command(tempBanCommandName).Triggers()[0])))
		return
	}

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			c.Replyf(e, "Missing required permissions to temporarily ban users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		c.irc.Ban(channel, mask)

		task := models.NewBanRemovalTask(time.Now().Add(seconds), mask, channel)
		err = firestore.Get().AddTask(task)
		if err != nil {
			logger.Errorf(e, "error adding task, %s", err)
			return
		}

		logger.Infof(e, "temporarily banned %s from %s for %s", mask, channel, duration)
	})
}
