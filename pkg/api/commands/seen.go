package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const SeenCommandName = "seen"

type SeenCommand struct {
	*commandStub
}

func NewSeenCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &SeenCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *SeenCommand) Name() string {
	return SeenCommandName
}

func (c *SeenCommand) Description() string {
	return "Shows the last message sent by the specified user."
}

func (c *SeenCommand) Triggers() []string {
	return []string{"seen"}
}

func (c *SeenCommand) Usages() []string {
	return []string{"%s <nick>"}
}

func (c *SeenCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *SeenCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *SeenCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())

	tokens := Tokens(e.Message())
	nick := tokens[1]

	u, err := repository.GetUserByNick(e, e.ReplyTarget(), nick, false)
	if err != nil {
		c.Replyf(e, "I ran into an error looking for %s.", style.Bold(nick))
		return
	}

	if u == nil || len(u.RecentMessages) == 0 {
		c.Replyf(e, "Sorry, I haven't seen %s.", style.Bold(nick))
		return
	}

	lastMessage := u.RecentMessages[len(u.RecentMessages)-1]
	c.Replyf(e, "%s was last seen %s saying: %s", style.Bold(nick), style.Bold(elapse.PastTimeDescription(lastMessage.At)), style.Italics(lastMessage.Message))
}
