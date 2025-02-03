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

const TempMuteCommandName = "tempmute"

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
	return []string{"%s <duration> <nick>"}
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

			if len(reason) == 0 {
				reason = fmt.Sprintf("temporarily muted for %s", elapse.ParseDurationDescription(duration))
			} else {
				reason = fmt.Sprintf("%s - temporarily muted for %s", reason, elapse.ParseDurationDescription(duration))
			}

			ch, err := repository.GetChannel(e, channel)
			if err != nil {
				logger.Errorf(e, "error retrieving channel, %s", err)
				return
			}

			users, err := repository.GetAllUsersMatchingUserHost(e, channel, nick)
			if err != nil {
				logger.Errorf(e, "error getting users by host: %v", err)
				return
			}

			if len(users) > 0 {
				nicks := make([]string, len(users))
				for _, u := range users {
					nicks = append(nicks, u.Nick)
				}
				logger.Debugf(e, "users matching host %s: %v", nick, strings.Join(nicks, ", "))
			}

			var specifiedUser *models.User
			for _, u := range users {
				if u.Nick == nick {
					specifiedUser = u
					break
				}
			}

			// some users don't yet have a host populated, so find them by nick and add them to users slice
			if specifiedUser == nil {
				logger.Debugf(e, "specified user not found by host, retrieving by nick")
				specifiedUser, err = repository.GetUserByNick(e, channel, nick, false)
				if err != nil {
					logger.Errorf(e, "error retrieving user by nick, %s", err)
					return
				}
				users = append(users, specifiedUser)
			}

			isAutoVoiced := repository.IsChannelAutoVoicedUser(e, ch, nick) || (specifiedUser != nil && specifiedUser.IsAutoVoiced)
			logger.Debugf(e, "isAutoVoiced? %t", isAutoVoiced)

			c.Replyf(e, "Temporarily muted %s for %s.", style.Bold(nick), style.Bold(elapse.ParseDurationDescription(duration)))

			go func() {
				c.irc.Mute(channel, nick)
				logger.Debugf(e, "muted %s in %s", nick, channel)

				if isAutoVoiced {
					logger.Debugf(e, "removing channel auto-voiced user %s", nick)
					repository.RemoveChannelAutoVoicedUser(e, ch, nick)
					if err = repository.UpdateChannelAutoVoiced(e, ch); err != nil {
						logger.Errorf(e, "error updating channel, %s", err)
						return
					}

					logger.Debugf(e, "removing auto-voice from %d users records", len(users))
					for _, u := range users {
						logger.Debugf(e, "removing auto-voice from %s (%s)", u.Nick, u.Host)
						u.IsAutoVoiced = false
						if err = repository.UpdateUserIsAutoVoiced(e, u); err != nil {
							logger.Errorf(e, "error updating user isAutoVoiced, %s", err)
						} else {
							logger.Debugf(e, "removed auto-voice from %s", u.Nick)
						}
					}
				}

				logger.Debugf(e, "adding task to remove mute from %s in %s in %s", nick, channel, elapse.ParseDurationDescription(duration))
				for _, u := range users {
					logger.Debugf(e, "adding mute removal task for %s (%s)", u.Nick, u.Host)
					task := models.NewMuteRemovalTask(time.Now().Add(seconds), u.Nick, channel, isAutoVoiced)
					err = firestore.Get().AddTask(task)
					if err != nil {
						logger.Errorf(e, "error adding task, %s", err)
						continue
					}
					logger.Infof(e, "temporarily muted %s from %s for %s", nick, channel, elapse.ParseDurationDescription(duration))
				}
			}()
		})
	})
}
