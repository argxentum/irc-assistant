package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"slices"
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
	return "Mutes the specified user in the channel. If they were previously auto-voiced, they will be removed from the auto-voice list."
}

func (c *MuteCommand) Triggers() []string {
	return []string{"mute", "m"}
}

func (c *MuteCommand) Usages() []string {
	return []string{"%s <nick>"}
}

func (c *MuteCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *MuteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *MuteCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	nick := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to mute users in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		fs := firestore.Get()

		ch, err := fs.Channel(channel)
		if err != nil {
			logger.Errorf(e, "error retrieving channel, %s", err)
			return
		}

		if ch.AutoVoiced != nil && slices.Contains(ch.AutoVoiced, nick) {
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

		c.irc.Mute(channel, nick)
		logger.Infof(e, "muted %s in %s", nick, channel)
	})
}
