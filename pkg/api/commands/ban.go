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
	"fmt"
	"strings"
	"time"
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

	if len(duration) > 0 {
		if len(reason) == 0 {
			reason = fmt.Sprintf("temporarily banned for %s", elapse.ParseDurationDescription(duration))
		} else {
			reason = fmt.Sprintf("%s - temporarily banned for %s", reason, elapse.ParseDurationDescription(duration))
		}
	}

	c.authorizer.ListUsersByMask(channel, mask, func(users []*irc.User) {
		time.Sleep(250 * time.Millisecond)
		for _, user := range users {
			c.irc.Kick(channel, user.Mask.Nick, reason)
			logger.Infof(e, "kicked %s from %s: %s", mask, channel, reason)
			time.Sleep(25 * time.Millisecond)
		}
	})

	c.irc.Ban(channel, mask)

	if len(duration) > 0 {
		go func() {
			seconds, err := elapse.ParseDuration(duration)
			if err != nil {
				logger.Errorf(e, "error parsing duration, %s", err)
				c.Replyf(e, "invalid duration, see %s for help", style.Italics(fmt.Sprintf("%s%s", c.cfg.Commands.Prefix, c.Triggers()[0])))
				return
			}

			m := irc.ParseMask(mask)
			task := models.NewBanRemovalTask(time.Now().Add(seconds), m.String(), channel)
			err = firestore.Get().AddTask(task)
			if err != nil {
				logger.Errorf(e, "error adding task, %s", err)
				return
			}
		}()
	}

	logger.Infof(e, "banned %s in %s", mask, channel)
}
