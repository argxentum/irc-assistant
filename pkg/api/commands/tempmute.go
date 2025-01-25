package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"slices"
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

	duration := strings.Replace(tokens[1], "+", "", 1)
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

			fs := firestore.Get()
			ch, err := fs.Channel(channel)
			if err != nil {
				logger.Errorf(e, "error retrieving channel, %s", err)
				return
			}

			if ch == nil {
				logger.Errorf(e, "channel %s does not exist", channel)
				return
			}

			isAutoVoiced := ch.AutoVoiced != nil && slices.Contains(ch.AutoVoiced, nick)
			c.Replyf(e, "Temporarily muted %s for %s.", style.Bold(nick), style.Bold(elapse.ParseDurationDescription(duration)))

			go func() {
				c.irc.Mute(channel, nick)

				if isAutoVoiced {
					voiced := make([]string, 0)
					for _, n := range ch.AutoVoiced {
						if n != nick {
							voiced = append(voiced, n)
						}
					}
					ch.AutoVoiced = voiced

					if err = fs.UpdateChannel(ch.Name, map[string]interface{}{"auto_voiced": ch.AutoVoiced}); err != nil {
						logger.Errorf(e, "error updating channel, %s", err)
						return
					}
				}

				task := models.NewMuteRemovalTask(time.Now().Add(seconds), nick, channel, isAutoVoiced)
				err = fs.AddTask(task)
				if err != nil {
					logger.Errorf(e, "error adding task, %s", err)
					return
				}

				logger.Infof(e, "temporarily muted %s from %s for %s", nick, channel, elapse.ParseDurationDescription(duration))
			}()
		})
	})
}
