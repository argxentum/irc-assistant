package commands

import (
	"assistant/pkg/api/actions"
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const BanCommandName = "ban"

type BanCommand struct {
	*commandStub
}

func NewBanCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &BanCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *BanCommand) Name() string {
	return BanCommandName
}

func (c *BanCommand) Description() string {
	return "Kicks and bans the given user mask from the channel. If a duration is specified, it will be a temporary ban."
}

func (c *BanCommand) Triggers() []string {
	return []string{"ban", "b", "kb", "tb"}
}

func (c *BanCommand) Usages() []string {
	return []string{"%s [<duration>] <mask> [<reason>]"}
}

func (c *BanCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *BanCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *BanCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	channel := e.ReplyTarget()
	tokens := Tokens(e.Message())

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "lacking needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions for %s command in this channel. Did you forget /mode %s +h %s?", style.Bold(c.Triggers()[0]), channel, c.cfg.IRC.Nick)
			return
		}

		var mask, duration, reason string

		// attempt to correct for accidentally swapping mask/duration if issuing a temp ban
		if len(tokens) > 2 {
			if elapse.IsDuration(tokens[1]) {
				duration = tokens[1]
				mask = tokens[2]
			} else if elapse.IsDuration(tokens[2]) {
				mask = tokens[1]
				duration = tokens[2]
			}
		}

		if len(mask) == 0 {
			mask = tokens[1]
		}

		reasonIdx := 2
		if len(duration) > 0 {
			reasonIdx++
		}

		if len(tokens) > reasonIdx {
			reason = strings.Join(tokens[reasonIdx:], " ")
		}

		logger.Infof(e, "⚡ %s [%s/%s] %s %s %s", c.Name(), e.From, e.ReplyTarget(), channel, mask, duration)

		c.ban(e, channel, mask, duration, reason)
	})
}

func (c *BanCommand) ban(e *irc.Event, channel, mask, duration, reason string) {
	logger := log.Logger()

	// if mask is a plain nick (no ! or @), resolve to *!*@host
	if !strings.Contains(mask, "!") && !strings.Contains(mask, "@") {
		done := make(chan *irc.User, 1)
		c.authorizer.GetUser(channel, mask, func(user *irc.User) {
			done <- user
		})
		user := <-done

		if user == nil {
			c.Replyf(e, "%s not found in channel", mask)
			return
		}

		nick := mask
		mask = fmt.Sprintf("*!*@%s", user.Mask.Host)
		logger.Debugf(e, "resolved %s to %s", nick, mask)
	}

	actions.Ban(c.irc, channel, mask, duration, reason)
}
