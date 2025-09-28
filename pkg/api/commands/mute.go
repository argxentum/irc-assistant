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
	return "Mutes the specified user in the channel and removes auto-voice, if applicable. If duration is specified, the user will be temporarily muted for that duration."
}

func (c *MuteCommand) Triggers() []string {
	return []string{"mute", "m", "tm"}
}

func (c *MuteCommand) Usages() []string {
	return []string{"%s [<duration>] <nick> [<reason>]"}
}

func (c *MuteCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *MuteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *MuteCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()

	var nick, duration, reason string

	// attempt to correct for accidentally swapping nick/duration if issuing a temp ban
	if len(tokens) > 2 {
		if elapse.IsDuration(tokens[1]) {
			duration = tokens[1]
			nick = tokens[2]
		} else if elapse.IsDuration(tokens[2]) {
			nick = tokens[1]
			duration = tokens[2]
		}
	}

	if len(nick) == 0 {
		nick = tokens[1]
	}

	reasonIdx := 2
	if len(duration) > 0 {
		reasonIdx++
	}

	if len(tokens) > reasonIdx {
		reason = strings.Join(tokens[reasonIdx:], " ")
	}

	c.mute(e, channel, nick, duration, reason)
}

func (c *MuteCommand) mute(e *irc.Event, channel, nick, duration, reason string) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	var seconds time.Duration
	if len(duration) > 0 {
		var err error
		seconds, err = elapse.ParseDuration(duration)
		if err != nil {
			logger.Errorf(e, "error parsing duration, %s", err)
			c.Replyf(e, "invalid duration, see %s for help", style.Bold(fmt.Sprintf("%s%s", c.cfg.Commands.Prefix, registry.Command(MuteCommandName).Triggers()[0])))
			return
		}
	}

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			c.Replyf(e, "Missing required permissions to mute users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		c.authorizer.GetUser(e.ReplyTarget(), nick, func(iu *irc.User) {
			if iu == nil {
				c.Replyf(e, "User %s not found", style.Bold(nick))
				return
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
			user, err := repository.GetUserByNick(e, channel, nick, true)
			if err != nil {
				logger.Errorf(e, "error retrieving user by nick, %s", err)
				return
			}

			if user == nil {
				logger.Errorf(e, "user %s not found by nick", nick)
				return
			}

			users := make([]*models.User, 0)
			users = append(users, user)

			// get all users with the same host
			if len(user.Host) > 0 {
				hus, err := repository.GetUsersByHost(e, channel, iu.Mask.Host)
				if err != nil {
					logger.Errorf(e, "error getting users by host: %v", err)
					return
				}

				for _, hu := range hus {
					if hu.Nick != user.Nick {
						users = append(users, hu)
					}
				}
			}

			go func() {
				var msg string
				if seconds > 0 {
					msg = fmt.Sprintf("\U0001F507 Temporarily muting %s for %s", style.Bold(nick), style.Bold(elapse.ParseDurationDescription(duration)))
				} else {
					msg = fmt.Sprintf("\U0001F507 Muting %s", style.Bold(nick))
				}
				if len(reason) > 0 {
					msg += ": " + reason
				}

				c.SendMessage(e, e.ReplyTarget(), msg)
				isAutoVoiced := repository.IsChannelAutoVoicedUser(e, ch, nick) || user.IsAutoVoiced

				// mute and remove auto-voice from all users
				for _, u := range users {
					c.irc.Mute(channel, u.Nick)

					if seconds > 0 {
						logger.Infof(e, "temporarily muted %s in %s for %s", nick, channel, duration)
					} else {
						logger.Infof(e, "muted %s in %s", nick, channel)
					}

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

				if seconds > 0 {
					// only need to add a single task for unmuting, since it will unmute all users with a matching host
					logger.Debugf(e, "adding task to unmute from %s in %s in %s", nick, channel, duration)
					task := models.NewMuteRemovalTask(time.Now().Add(seconds), channel, nick, iu.Mask.Host, isAutoVoiced)
					err = firestore.Get().AddTask(task)
					if err != nil {
						logger.Errorf(e, "error adding task, %s", err)
						return
					}
				}
			}()
		})
	})
}
