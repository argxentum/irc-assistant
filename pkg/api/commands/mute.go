package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
)

const MuteCommandName = "mute"

type MuteCommand struct {
	*commandStub
}

func NewMuteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &MuteCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *MuteCommand) Name() string {
	return MuteCommandName
}

func (c *MuteCommand) Description() string {
	return "Mutes the specified user in the channel."
}

func (c *MuteCommand) Triggers() []string {
	return []string{"mute", "m"}
}

func (c *MuteCommand) Usages() []string {
	return []string{"%s <nick>"}
}

func (c *MuteCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *MuteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *MuteCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	nick := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to mute users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		ch, err := repository.GetChannel(e, channel)
		if err != nil {
			logger.Errorf(e, "error retrieving channel, %s", err)
			return
		}

		if ch == nil {
			logger.Warningf(e, "channel not found: %s", channel)
			return
		}

		users := make([]*models.User, 0)

		u, err := repository.GetUserByNick(e, channel, nick, true)
		if err != nil {
			logger.Errorf(e, "error getting users by host: %v", err)
			return
		}

		users = append(users, u)

		if len(u.Host) > 0 {
			hus, err := repository.GetUsersByHost(e, channel, u.Host)
			if err != nil {
				logger.Errorf(e, "error getting users by host: %v", err)
				return
			}

			for _, hu := range hus {
				if hu.Nick != u.Nick {
					users = append(users, hu)
				}
			}
		}

		for _, user := range users {
			c.irc.Mute(channel, user.Nick)
			logger.Infof(e, "muted %s in %s", nick, channel)

			repository.RemoveChannelAutoVoicedUser(e, ch, nick)
			if err = repository.UpdateChannelAutoVoiced(e, ch); err != nil {
				logger.Errorf(e, "error updating channel, %s", err)
				return
			}

			user.IsAutoVoiced = false
			if err = repository.UpdateUserIsAutoVoiced(e, channel, user); err != nil {
				logger.Errorf(e, "error updating user isAutoVoiced, %s", err)
			}
		}
	})
}
