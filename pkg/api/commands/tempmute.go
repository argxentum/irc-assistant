package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"strings"
	"time"
)

const TempMuteCommandName = "temp_mute"

type TempMuteCommand struct {
	*commandStub
}

func NewTempMuteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &TempMuteCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *TempMuteCommand) Name() string {
	return TempMuteCommandName
}

func (c *TempMuteCommand) Description() string {
	return "Temporarily mutes the specified user in the channel for the specified duration."
}

func (c *TempMuteCommand) Triggers() []string {
	return []string{"tempmute", "tm"}
}

func (c *TempMuteCommand) Usages() []string {
	return []string{"%s <duration> <nick> [<reason>]"}
}

func (c *TempMuteCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *TempMuteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 2)
}

func (c *TempMuteCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()

	duration := tokens[1]
	nick := tokens[2]

	reason := ""
	if len(tokens) > 3 {
		reason = strings.Join(tokens[3:], " ")
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	seconds, err := elapse.ParseDuration(duration)
	if err != nil {
		logger.Errorf(e, "error parsing duration, %s", err)
		c.Replyf(e, "invalid duration, see %s for help", style.Bold(fmt.Sprintf("%s%s", c.cfg.Commands.Prefix, registry.Command(TempMuteCommandName).Triggers()[0])))
		return
	}

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			c.Replyf(e, "Missing required permissions to temporarily mute users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		c.authorizer.GetUser(e.ReplyTarget(), nick, func(user *irc.User) {
			if user == nil {
				c.Replyf(e, "User %s not found", style.Bold(nick))
				return
			}

			msg := fmt.Sprintf("\U0001F507 Temporarily muting %s for %s", style.Bold(nick), style.Bold(elapse.ParseDurationDescription(duration)))
			if len(reason) > 0 {
				msg = fmt.Sprintf("\U0001F507 Temporarily muting %s for %s: %s", style.Bold(nick), style.Bold(elapse.ParseDurationDescription(duration)), reason)
			}

			ch, err := repository.GetChannel(e, channel)
			if err != nil {
				logger.Errorf(e, "error retrieving channel, %s", err)
				return
			}

			if ch == nil {
				logger.Errorf(e, "channel %s not found", channel)
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
				c.SendMessage(e, e.ReplyTarget(), msg)
				isAutoVoiced := repository.IsChannelAutoVoicedUser(e, ch, nick) || u.IsAutoVoiced

				//mute and remove auto-voice from all users
				for _, u := range users {
					c.irc.Mute(channel, u.Nick)
					logger.Infof(e, "temporarily muted %s from %s for %s", nick, channel, elapse.ParseDurationDescription(duration))

					repository.RemoveChannelAutoVoicedUser(e, ch, u.Nick)
					if err = repository.UpdateChannelAutoVoiced(e, ch); err != nil {
						logger.Errorf(e, "error updating channel, %s", err)
						return
					}
					logger.Debugf(e, "removed channel auto-voice from user %s", u.Nick)

					u.IsAutoVoiced = false
					if err = repository.UpdateUserIsAutoVoiced(e, channel, u); err != nil {
						logger.Errorf(e, "error updating user isAutoVoiced, %s", err)
					}
					logger.Debugf(e, "removed auto-voice from user %s", u.Nick)
				}

				// only need to add a single task for unmuting, since it will unmute all users with a matching host
				logger.Debugf(e, "adding task to unmute from %s in %s in %s", nick, channel, elapse.ParseDurationDescription(duration))
				task := models.NewMuteRemovalTask(time.Now().Add(seconds), channel, nick, user.Mask.Host, isAutoVoiced)
				err = firestore.Get().AddTask(task)
				if err != nil {
					logger.Errorf(e, "error adding task, %s", err)
					return
				}
			}()
		})
	})
}
