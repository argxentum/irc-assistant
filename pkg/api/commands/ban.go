package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const BanCommandName = "ban"

type BanCommand struct {
	*commandStub
}

func NewBanCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &BanCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *BanCommand) Name() string {
	return BanCommandName
}

func (c *BanCommand) Description() string {
	return "Bans the given user mask from the channel. If a duration is specified, it will be a temporary ban; otherwise, the ban is permanent."
}

func (c *BanCommand) Triggers() []string {
	return []string{"ban", "b"}
}

func (c *BanCommand) Usages() []string {
	return []string{"%s [<duration>] <mask> [<reason>]"}
}

func (c *BanCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *BanCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *BanCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	channel := e.ReplyTarget()
	tokens := Tokens(e.Message())

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to ban users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		// check if user is trying to issue a temp ban, which uses the syntax: !ban <duration> <nick> [<reason>]
		if len(tokens) > 2 && elapse.IsDuration(tokens[1]) {
			c.authorizer.GetUser(channel, tokens[2], func(user *irc.User) {
				// if the second token is a duration and the third token is a user found in the channel, treat as temp ban
				if user != nil {
					registry.Command(TempBanCommandName).Execute(e)
				} else {
					c.ban(e, channel, tokens[1])
				}
			})
		} else {
			c.ban(e, channel, tokens[1])
		}
	})
}

func (c *BanCommand) ban(e *irc.Event, channel, mask string) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, mask)
	tokens := Tokens(e.Message())

	// if mask is actually a user nick, also kick if the user is in the channel
	c.authorizer.GetUser(channel, mask, func(user *irc.User) {
		reason := ""
		if len(tokens) > 2 {
			reason = strings.Join(tokens[2:], " ")
		}

		if user != nil {
			c.irc.Kick(channel, mask, reason)
			logger.Infof(e, "kicked %s from %s: %s", mask, channel, reason)
		}
	})

	c.irc.Ban(channel, mask)
	logger.Infof(e, "banned %s in %s", mask, channel)
}
