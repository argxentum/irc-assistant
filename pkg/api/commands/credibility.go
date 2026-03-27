package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
)

const CredibilityCommandName = "credibility"

type CredibilityCommand struct {
	*commandStub
}

func NewCredibilityCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &CredibilityCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *CredibilityCommand) Name() string {
	return CredibilityCommandName
}

func (c *CredibilityCommand) Description() string {
	return "Displays the credibility score for a user based on the sources they share."
}

func (c *CredibilityCommand) Triggers() []string {
	return []string{"credibility"}
}

func (c *CredibilityCommand) Usages() []string {
	return []string{"%s <user>"}
}

func (c *CredibilityCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *CredibilityCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *CredibilityCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()
	nick := tokens[1]

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), nick)

	u, err := repository.GetUserByNick(e, channel, nick, false)
	if err != nil {
		logger.Errorf(e, "error getting user, %s", err)
		c.Replyf(e, "unable to get credibility for %s.", style.Bold(nick))
		return
	}

	if u == nil {
		c.Replyf(e, "no credibility data found for %s.", style.Bold(nick))
		return
	}

	total := u.HighCredibilityCount + u.LowCredibilityCount
	if total == 0 {
		c.Replyf(e, "no credibility data found for %s.", style.Bold(nick))
		return
	}

	score := float64(u.HighCredibilityCount) / float64(total) * 100

	scoreColor := style.ColorGreen
	if score < 50 {
		scoreColor = style.ColorRed
	} else if score < 75 {
		scoreColor = style.ColorYellow
	}

	c.SendMessage(e, channel, fmt.Sprintf("%s has a credibility score of %s",
		style.Bold(nick),
		style.Bold(style.ColorForeground(fmt.Sprintf("%.0f%%", score), scoreColor)),
	))
}
