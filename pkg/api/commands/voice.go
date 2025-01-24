package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
)

const AutoVoiceCommandName = "auto_voice"

type AutoVoiceCommand struct {
	*commandStub
}

func NewAutoVoiceCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &AutoVoiceCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *AutoVoiceCommand) Name() string {
	return AutoVoiceCommandName
}

func (c *AutoVoiceCommand) Description() string {
	return "Adds the specified user to the auto-voice list for the channel."
}

func (c *AutoVoiceCommand) Triggers() []string {
	return []string{"autovoice", "v"}
}

func (c *AutoVoiceCommand) Usages() []string {
	return []string{"%s <nick>"}
}

func (c *AutoVoiceCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *AutoVoiceCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *AutoVoiceCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	nick := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to auto-voice users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		fs := firestore.Get()
		ch, err := fs.Channel(e.ReplyTarget())
		if err != nil {
			logger.Errorf(e, "error retrieving channel, %s", err)
			return
		}

		if ch == nil {
			logger.Errorf(e, "channel %s does not exist", channel)
			return
		}

		if ch.AutoVoiced == nil {
			ch.AutoVoiced = make([]string, 0)
		}

		ch.AutoVoiced = append(ch.AutoVoiced, nick)
		if err = fs.UpdateChannel(ch.Name, map[string]any{"auto_voiced": ch.AutoVoiced}); err != nil {
			logger.Errorf(e, "error updating channel, %s", err)
			return
		}

		c.irc.Voice(channel, nick)
		logger.Infof(e, "voiced %s in %s", nick, channel)
	})
}
