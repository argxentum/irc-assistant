package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
)

const UnmuteCommandName = "unmute"

type UnmuteCommand struct {
	*commandStub
}

func NewUnmuteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &UnmuteCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *UnmuteCommand) Name() string {
	return UnmuteCommandName
}

func (c *UnmuteCommand) Description() string {
	return "Unmutes the specified user in the channel. If no channel is specified, the current channel is used."
}

func (c *UnmuteCommand) Triggers() []string {
	return []string{"unmute", "um"}
}

func (c *UnmuteCommand) Usages() []string {
	return []string{"%s <nick> [<channel>]"}
}

func (c *UnmuteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *UnmuteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *UnmuteCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	nick := tokens[1]

	channel := e.ReplyTarget()
	if len(tokens) > 2 {
		channel = tokens[2]
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to unmute users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
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

		if ch.VoiceRequests == nil {
			ch.VoiceRequests = make([]models.VoiceRequest, 0)
		}

		voiceRequests := make([]models.VoiceRequest, 0)
		for _, request := range ch.VoiceRequests {
			if request.Nick != nick {
				voiceRequests = append(voiceRequests, request)
			}
		}

		ch.VoiceRequests = voiceRequests
		if err = fs.UpdateChannel(ch.Name, map[string]any{"voice_requests": ch.VoiceRequests}); err != nil {
			logger.Errorf(e, "error updating channel, %s", err)
			return
		}

		c.irc.Voice(channel, nick)
		logger.Infof(e, "unmuted %s in %s", nick, channel)
	})
}
