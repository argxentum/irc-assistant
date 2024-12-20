package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const KickCommandName = "kick"

type KickCommand struct {
	*commandStub
}

func NewKickCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &KickCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *KickCommand) Name() string {
	return KickCommandName
}

func (c *KickCommand) Description() string {
	return "Kicks the specified user from the channel."
}

func (c *KickCommand) Triggers() []string {
	return []string{"kick", "k"}
}

func (c *KickCommand) Usages() []string {
	return []string{"%s <nick> [<reason>]"}
}

func (c *KickCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *KickCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *KickCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	nick := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to kick users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		reason := ""
		if len(tokens) > 2 {
			reason = strings.Join(tokens[2:], " ")
		}
		c.irc.Kick(channel, nick, reason)
		logger.Infof(e, "kicked %s in %s", nick, channel)
	})
}
