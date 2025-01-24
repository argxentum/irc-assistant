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
	"cmp"
	"fmt"
	"slices"
)

const VoiceRequestsCommandName = "voice_requests"

type VoiceRequestsCommand struct {
	*commandStub
}

func NewVoiceRequestsCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &VoiceRequestsCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *VoiceRequestsCommand) Name() string {
	return VoiceRequestsCommandName
}

func (c *VoiceRequestsCommand) Description() string {
	return "Shows voice requests for the specified channel. If no channel is specified, the current channel is used."
}

func (c *VoiceRequestsCommand) Triggers() []string {
	return []string{"voicerequests", "vr"}
}

func (c *VoiceRequestsCommand) Usages() []string {
	return []string{"%s [<channel>]"}
}

func (c *VoiceRequestsCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *VoiceRequestsCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *VoiceRequestsCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if e.IsPrivateMessage() {
		if len(tokens) > 1 {
			channel = tokens[1]
		} else {
			c.Replyf(e, "Please specify a channel to view voice requests for: %s <channel>", c.Name())
			return
		}
	}

	nick := e.From
	if e.IsPrivateMessage() {
		nick = e.ReplyTarget()
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to view voice requests in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		fs := firestore.Get()
		ch, err := fs.Channel(channel)
		if err != nil {
			logger.Errorf(e, "error retrieving channel, %s", err)
			return
		}

		if ch.VoiceRequests == nil {
			ch.VoiceRequests = make([]models.VoiceRequest, 0)
		}

		slices.SortFunc(ch.VoiceRequests, func(a, b models.VoiceRequest) int {
			return cmp.Compare(a.RequestedAt.Unix(), b.RequestedAt.Unix())
		})

		messages := make([]string, 0)
		messages = append(messages, fmt.Sprintf("%s voice requests in %s", style.Bold(fmt.Sprintf("%d", len(ch.VoiceRequests))), style.Bold(channel)))

		for _, vr := range ch.VoiceRequests {
			messages = append(messages, fmt.Sprintf("%s, %s", vr.Nick, elapse.PastTimeDescription(vr.RequestedAt)))
		}

		c.SendMessages(e, e.ReplyTarget(), messages)

		logger.Infof(e, "unmuted %s in %s", nick, channel)
	})
}
