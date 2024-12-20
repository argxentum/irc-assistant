package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const UnbanCommandName = "unban"

type UnbanCommand struct {
	*commandStub
}

func NewUnbanCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &UnbanCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *UnbanCommand) Name() string {
	return UnbanCommandName
}

func (c *UnbanCommand) Description() string {
	return "Unbans the given user mask from the channel."
}

func (c *UnbanCommand) Triggers() []string {
	return []string{"unban", "ub"}
}

func (c *UnbanCommand) Usages() []string {
	return []string{"%s <mask>"}
}

func (c *UnbanCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *UnbanCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *UnbanCommand) Execute(e *irc.Event) {
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
