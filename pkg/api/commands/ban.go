package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const banCommandName = "ban"

type banCommand struct {
	*commandStub
}

func NewBanCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &banCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *banCommand) Name() string {
	return banCommandName
}

func (c *banCommand) Description() string {
	return "Bans the given user mask from the channel."
}

func (c *banCommand) Triggers() []string {
	return []string{"ban", "b"}
}

func (c *banCommand) Usages() []string {
	return []string{"%s <mask>"}
}

func (c *banCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *banCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *banCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	mask := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, mask)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to ban users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		c.irc.Ban(channel, mask)
		logger.Infof(e, "banned %s in %s", mask, channel)
	})
}
