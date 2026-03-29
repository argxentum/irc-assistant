package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/penalty"
	"fmt"
)

const UserPenaltiesName = "user_penalties"

type UserPenalties struct {
	*commandStub
}

func NewUserPenaltiesCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &UserPenalties{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *UserPenalties) Name() string {
	return UserPenaltiesName
}

func (c *UserPenalties) Description() string {
	return "Shows current penalties for a channel user."
}

func (c *UserPenalties) Triggers() []string {
	return []string{"penalties", "ps"}
}

func (c *UserPenalties) Usages() []string {
	return []string{"%s [<channel>] <nick>"}
}

func (c *UserPenalties) AllowedInPrivateMessages() bool {
	return true
}

func (c *UserPenalties) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *UserPenalties) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if len(tokens) > 2 && irc.IsChannel(tokens[1]) {
		channel = tokens[1]
		tokens = append(tokens[:1], tokens[2:]...)
	}

	nick := tokens[1]

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), channel)

	user, err := repository.GetUserByNick(e, channel, nick, false)
	if err != nil {
		logger.Errorf(e, "failed to get user %s in channel %s: %v", nick, channel, err)
		c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("An error occurred while retrieving penalties for %s in %s.", style.Bold(nick), channel))
		return
	}

	if user == nil {
		c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("User %s not found in channel %s.", style.Bold(nick), channel))
		return
	}

	status := penalty.Calculate(user.Penalty, user.ExtendedPenalty, c.cfg.DisinfoPenalty)

	mutePctStr := fmt.Sprintf("%.0f%%", status.MutePercent)
	if status.MutePercent >= 75 {
		mutePctStr = style.ColorForeground(mutePctStr, style.ColorRed)
	} else if status.MutePercent >= 50 {
		mutePctStr = style.ColorForeground(mutePctStr, style.ColorYellow)
	}

	banPctStr := fmt.Sprintf("%.0f%%", status.BanPercent)
	if status.BanPercent >= 75 {
		banPctStr = style.ColorForeground(banPctStr, style.ColorRed)
	} else if status.BanPercent >= 50 {
		banPctStr = style.ColorForeground(banPctStr, style.ColorYellow)
	}

	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("Penalty status for %s in %s • temporary mute: %s • temporary ban: %s", style.Bold(nick), channel, mutePctStr, banPctStr))
}
