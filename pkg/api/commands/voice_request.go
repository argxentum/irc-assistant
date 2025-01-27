package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"time"
)

const VoiceRequestCommandName = "voice_request"

const (
	voiceRequestNotificationEach      = "each"
	voiceRequestNotificationScheduled = "scheduled"
)

type VoiceRequestCommand struct {
	*commandStub
}

func NewVoiceRequestCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &VoiceRequestCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *VoiceRequestCommand) Name() string {
	return VoiceRequestCommandName
}

func (c *VoiceRequestCommand) Description() string {
	return "Requests voice (+v) in the specified channel."
}

func (c *VoiceRequestCommand) Triggers() []string {
	return []string{"voice"}
}

func (c *VoiceRequestCommand) Usages() []string {
	return []string{"%s <channel>"}
}

func (c *VoiceRequestCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *VoiceRequestCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *VoiceRequestCommand) Execute(e *irc.Event) {
	if !e.IsPrivateMessage() {
		return
	}

	tokens := Tokens(e.Message())
	channel := tokens[1]
	nick := e.ReplyTarget()
	mask := irc.Parse(e.Source)

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	ch, err := repository.GetChannel(e, channel)
	if err != nil {
		logger.Errorf(e, "error retrieving channel, %s", err)
		return
	}

	if repository.VoiceRequestExistsForNick(e, ch, mask.Nick) {
		c.Replyf(e, "You've already requested voice in %s. We'll review your request as soon as possible. Thanks for your patience.", channel)
		logger.Debugf(e, "voice already requested %s in %s", nick, channel)
		return
	} else if repository.VoiceRequestExistsForHost(e, ch, mask.Host) {
		c.Replyf(e, "You've already requested voice in %s using a different nick. Please submit only one voice request at a time. We'll review your first request as soon as possible. Thanks for your patience.", channel)
		logger.Debugf(e, "voice already requested with different nick %s in %s", nick, channel)
		return
	}

	repository.AddChannelVoiceRequest(e, ch, mask)
	if err = repository.UpdateChannelVoiceRequests(e, ch); err != nil {
		logger.Errorf(e, "error updating channel, %s", err)
		return
	}

	c.Replyf(e, "Your voice request in %s has been received. We'll be in touch soon.", style.Bold(channel))

	if len(ch.VoiceRequestNotifications) > 0 {
		for _, vrn := range ch.VoiceRequestNotifications {
			if vrn.Interval == voiceRequestNotificationEach {
				c.irc.SendMessage(vrn.User, fmt.Sprintf("New voice request in %s: %s (%s)", channel, style.Bold(nick), mask))
			}
		}

		task := models.NewNotifyVoiceRequestsTask(nextNoonUTC(), channel)
		err = firestore.Get().AddTask(task)
		if err != nil {
			logger.Errorf(e, "error adding task, %s", err)
			return
		}
	}

	logger.Infof(e, "voice requested %s in %s", nick, channel)
}

func nextNoonUTC() time.Time {
	now := time.Now().UTC()
	todayNoon := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	if now.Hour() >= 12 {
		return todayNoon.AddDate(0, 0, 1)
	}
	return todayNoon
}
