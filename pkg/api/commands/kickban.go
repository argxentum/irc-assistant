package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
	"time"
)

const kickBanCommandName = "kickban"

type kickBanCommand struct {
	*commandStub
}

func NewKickBanCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &kickBanCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *kickBanCommand) Name() string {
	return kickBanCommandName
}

func (c *kickBanCommand) Description() string {
	return "Kicks and bans the specified user from the channel."
}

func (c *kickBanCommand) Triggers() []string {
	return []string{"kickban", "kb"}
}

func (c *kickBanCommand) Usages() []string {
	return []string{"%s <nick> [<reason>]"}
}

func (c *kickBanCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *kickBanCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *kickBanCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()
	nick := tokens[1]
	reason := ""
	if len(tokens) > 2 {
		reason = strings.Join(tokens[2:], " ")
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s - %s", c.Name(), e.From, e.ReplyTarget(), channel, nick, reason)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to kick users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		c.authorizer.GetUser(e.ReplyTarget(), nick, func(user *irc.User) {
			if user == nil {
				c.Replyf(e, "User %s not found", style.Bold(nick))
				return
			}

			go func() {
				c.irc.Ban(channel, user.Mask.NickWildcardString())
				time.Sleep(100 * time.Millisecond)
				c.irc.Kick(channel, nick, reason)

				logger.Infof(e, "kickBanned %s in %s", nick, channel)
			}()
		})
	})
}
