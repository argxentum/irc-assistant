package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const UnmuteCommandName = "unmute"

type UnmuteCommand struct {
	*commandStub
}

func NewUnmuteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &UnmuteCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *UnmuteCommand) Name() string {
	return UnmuteCommandName
}

func (c *UnmuteCommand) Description() string {
	return "Voices (unmutes) the specified user in the channel."
}

func (c *UnmuteCommand) Triggers() []string {
	return []string{"unmute", "um", "v"}
}

func (c *UnmuteCommand) Usages() []string {
	return []string{"%s <nick>"}
}

func (c *UnmuteCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *UnmuteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *UnmuteCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	nick := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to unmute users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		c.irc.Voice(channel, nick)
		logger.Infof(e, "unmuted %s in %s", nick, channel)
	})
}
