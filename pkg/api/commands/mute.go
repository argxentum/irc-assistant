package commands

import (
	"assistant/pkg/api/actions"
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
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
	return "Mutes the specified user in the channel and removes auto-voice, if applicable. If duration is specified, the user will be temporarily muted for that duration."
}

func (c *MuteCommand) Triggers() []string {
	return []string{"mute", "m", "tm"}
}

func (c *MuteCommand) Usages() []string {
	return []string{"%s [<channel>] [<duration>] <nick> [<reason>]"}
}

func (c *MuteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *MuteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *MuteCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())

	isGhostMute := false
	channel := e.ReplyTarget()
	if len(tokens) > 2 && irc.IsChannel(tokens[1]) {
		isGhostMute = true
		channel = tokens[1]
		tokens = append(tokens[:1], tokens[2:]...)
	}

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "lacking needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions for %s command in this channel. Did you forget /mode %s +h %s?", style.Bold(c.Triggers()[0]), channel, c.cfg.IRC.Nick)
			return
		}

		var nick, duration, reason string

		// attempt to correct for accidentally swapping nick/duration if issuing a temp mute
		if len(tokens) > 2 {
			if elapse.IsDuration(tokens[1]) {
				duration = tokens[1]
				nick = tokens[2]
			} else if elapse.IsDuration(tokens[2]) {
				nick = tokens[1]
				duration = tokens[2]
			}
		}

		if len(nick) == 0 {
			nick = tokens[1]
		}

		reasonIdx := 2
		if len(duration) > 0 {
			reasonIdx++
		}

		if len(tokens) > reasonIdx {
			reason = strings.Join(tokens[reasonIdx:], " ")
		}

		logger.Infof(e, "⚡ %s [%s/%s] %s %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick, duration)

		c.mute(e, channel, nick, duration, reason, isGhostMute)
	})
}

func (c *MuteCommand) mute(e *irc.Event, channel, nick, duration, reason string, isGhostMute bool) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	if isGhostMute {
		logger.Infof(e, "handling ghost mute of %s command in channel %s", nick, channel)
		c.irc.Mute(channel, nick)
		return
	}

	c.authorizer.GetUser(e.ReplyTarget(), nick, func(iu *irc.User) {
		if iu == nil {
			c.Replyf(e, "User %s not found", style.Bold(nick))
			return
		}

		go func() {
			actions.Mute(c.irc, channel, nick, iu.Mask.Host, duration, reason)
		}()
	})
}
