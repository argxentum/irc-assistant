package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const unbanCommandName = "unban"

type unbanCommand struct {
	*commandStub
}

func NewUnbanCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &unbanCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *unbanCommand) Name() string {
	return unbanCommandName
}

func (c *unbanCommand) Description() string {
	return "Unbans the given user mask from the channel."
}

func (c *unbanCommand) Triggers() []string {
	return []string{"unban", "ub"}
}

func (c *unbanCommand) Usages() []string {
	return []string{"%s <mask>"}
}

func (c *unbanCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *unbanCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *unbanCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	mask := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, mask)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to unban users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		c.irc.Unban(channel, mask)
		logger.Infof(e, "unbanned %s in %s", mask, channel)
	})
}
