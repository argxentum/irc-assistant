package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"strings"
	"time"
)

const KickBanCommandName = "kickban"

type KickBanCommand struct {
	*commandStub
}

func NewKickBanCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &KickBanCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *KickBanCommand) Name() string {
	return KickBanCommandName
}

func (c *KickBanCommand) Description() string {
	return "Kicks and bans the specified user from the channel."
}

func (c *KickBanCommand) Triggers() []string {
	return []string{"kickban", "kb"}
}

func (c *KickBanCommand) Usages() []string {
	return []string{"%s <nick> [<reason>]"}
}

func (c *KickBanCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *KickBanCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *KickBanCommand) Execute(e *irc.Event) {
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

			// get requested user by nick
			u, err := repository.GetUserByNick(e, channel, nick, true)
			if err != nil {
				logger.Errorf(e, "error retrieving user by nick, %s", err)
				return
			}

			if u == nil {
				logger.Errorf(e, "user %s not found by nick", nick)
				return
			}

			users := make([]*models.User, 0)
			users = append(users, u)

			// get all users with the same host
			if len(u.Host) > 0 {
				hus, err := repository.GetUsersByHost(e, channel, user.Mask.Host)
				if err != nil {
					logger.Errorf(e, "error getting users by host: %v", err)
					return
				}

				for _, hu := range hus {
					users = append(users, hu)
				}
			}

			go func() {
				//mute and remove auto-voice from all users
				for _, u := range users {
					c.irc.Mute(channel, u.Nick)

					u.IsAutoVoiced = false
					if err = repository.UpdateUserIsAutoVoiced(e, channel, u); err != nil {
						logger.Errorf(e, "error updating user isAutoVoiced, %s", err)
					}
					logger.Debugf(e, "removed auto-voice from user %s", u.Nick)
				}
			}()

			go func() {
				c.irc.Ban(channel, user.Mask.NickWildcardString())
				time.Sleep(100 * time.Millisecond)
				c.irc.Kick(channel, nick, reason)

				logger.Infof(e, "kickBanned %s in %s", nick, channel)
			}()
		})
	})
}
