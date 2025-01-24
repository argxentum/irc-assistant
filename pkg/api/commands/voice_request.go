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

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	fs := firestore.Get()
	ch, err := fs.Channel(channel)
	if err != nil {
		logger.Errorf(e, "error retrieving channel, %s", err)
		return
	}

	if ch.VoiceRequests == nil {
		ch.VoiceRequests = make([]models.VoiceRequest, 0)
	}

	if slices.ContainsFunc(ch.VoiceRequests, func(request models.VoiceRequest) bool { return request.Nick == nick }) {
		c.Replyf(e, "You have already requested voice in %s. We'll review your request as soon as possible. Thanks for your patience.", channel)
		logger.Debugf(e, "voice already requested %s in %s", nick, channel)
		return
	}

	ch.VoiceRequests = append(ch.VoiceRequests, models.VoiceRequest{Nick: nick, RequestedAt: time.Now()})
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
