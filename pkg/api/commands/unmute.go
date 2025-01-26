package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
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
	return "Unmutes the specified user in the channel."
}

func (c *UnmuteCommand) Triggers() []string {
	return []string{"unmute", "um"}
}

func (c *UnmuteCommand) Usages() []string {
	return []string{"%s [<channel>] <nick>"}
}

func (c *UnmuteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *UnmuteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *UnmuteCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())

	channel := ""
	nicks := make([]string, 0)

	if e.IsPrivateMessage() && irc.IsChannel(tokens[1]) && len(tokens) >= 3 {
		channel = tokens[1]
		nicks = tokens[2:]
	} else {
		channel = e.ReplyTarget()
		nicks = tokens[1:]
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, strings.Join(nicks, ", "))

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to unmute users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		ch, err := repository.GetChannel(e, channel)
		if err != nil {
			logger.Errorf(e, "error retrieving channel, %s", err)
			return
		}

		for _, nick := range nicks {
			repository.RemoveChannelVoiceRequest(e, ch, nick, "")
			c.irc.Voice(channel, nick)
		}

		if err = repository.UpdateChannelVoiceRequests(e, ch); err != nil {
			logger.Errorf(e, "error updating channel, %s", err)
			return
		}

		logger.Infof(e, "unmuted %s in %s", strings.Join(nicks, ", "), channel)
	})
}
