package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
)

const AutoVoiceCommandName = "auto_voice"

type AutoVoiceCommand struct {
	*commandStub
}

func NewAutoVoiceCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &AutoVoiceCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *AutoVoiceCommand) Name() string {
	return AutoVoiceCommandName
}

func (c *AutoVoiceCommand) Description() string {
	return "Adds the specified user to the auto-voice list for the channel."
}

func (c *AutoVoiceCommand) Triggers() []string {
	return []string{"autovoice", "v"}
}

func (c *AutoVoiceCommand) Usages() []string {
	return []string{"%s <nick>"}
}

func (c *AutoVoiceCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *AutoVoiceCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *AutoVoiceCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	nick := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to auto-voice users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
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

		repository.RemoveChannelVoiceRequest(e, ch, nick, "")
		if err = repository.UpdateChannelVoiceRequests(e, ch); err != nil {
			logger.Errorf(e, "error updating channel, %s", err)
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
			c.irc.Voice(channel, user.Nick)
			logger.Infof(e, "voiced %s (%s) in %s", user.Nick, user.Host, channel)

			user.IsAutoVoiced = true
			if err = repository.UpdateUserIsAutoVoiced(e, channel, user); err != nil {
				logger.Errorf(e, "error updating user isAutoVoiced, %s", err)
			}
		}
	})
}
