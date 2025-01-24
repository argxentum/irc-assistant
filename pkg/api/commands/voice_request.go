package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"slices"
	"time"
)

const VoiceRequestCommandName = "voice_request"

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

	if ch.VoiceRequests == nil {
		ch.VoiceRequests = make([]models.VoiceRequest, 0)
	}

	if slices.ContainsFunc(ch.VoiceRequests, func(request models.VoiceRequest) bool { return request.Nick == nick || request.Host == mask.Host }) {
		c.Replyf(e, "You have already requested voice in %s. We'll review your request as soon as possible. Thanks for your patience.", channel)
		logger.Debugf(e, "voice already requested %s in %s", nick, channel)
		return
	}

	vr := models.VoiceRequest{
		Nick:        mask.Nick,
		Username:    mask.UserID,
		Host:        mask.Host,
		RequestedAt: time.Now(),
	}

	ch.VoiceRequests = append(ch.VoiceRequests, vr)
	if err = fs.UpdateChannel(ch.Name, map[string]any{"voice_requests": ch.VoiceRequests}); err != nil {
		logger.Errorf(e, "error updating channel, %s", err)
		return
	}

	c.Replyf(e, "Your voice request in %s has been received. We'll be in touch soon.", style.Bold(channel))

	if c.cfg.IRC.MessageOwnerOnVoiceRequest {
		c.irc.SendMessage(c.cfg.IRC.Owner, fmt.Sprintf("New voice request in %s: %s", channel, nick))
	}

	logger.Infof(e, "voice requested %s in %s", nick, channel)
}
