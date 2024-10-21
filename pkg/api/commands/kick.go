package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const kickCommandName = "kick"

type kickCommand struct {
	*commandStub
}

func NewKickCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &kickCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *kickCommand) Name() string {
	return kickCommandName
}

func (c *kickCommand) Description() string {
	return "Kicks the specified user from the channel."
}

func (c *kickCommand) Triggers() []string {
	return []string{"kick", "k"}
}

func (c *kickCommand) Usages() []string {
	return []string{"%s <nick> [<reason>]"}
}

func (c *kickCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *kickCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *kickCommand) Execute(e *irc.Event) {
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
