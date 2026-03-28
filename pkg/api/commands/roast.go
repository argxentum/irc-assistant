package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
	"strings"
)

const RoastCommandName = "roast"

type RoastCommand struct {
	*commandStub
}

func NewRoastCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &RoastCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *RoastCommand) Name() string {
	return RoastCommandName
}

func (c *RoastCommand) Description() string {
	return "Roasts a user based on their recent messages"
}

func (c *RoastCommand) Triggers() []string {
	return []string{"roast"}
}

func (c *RoastCommand) Usages() []string {
	return []string{"%s <nick>"}
}

func (c *RoastCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *RoastCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *RoastCommand) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	nick := tokens[1]
	channel := e.ReplyTarget()

	logger.Infof(e, "⚡ %s [%s/%s] target: %s", c.Name(), e.From, channel, nick)

	fs := firestore.Get()
	user, err := fs.GetUserByNick(channel, nick)
	if err != nil {
		logger.Errorf(e, "error getting user %s: %s", nick, err)
		return
	}
	if user == nil || len(user.RecentMessages) == 0 {
		c.Replyf(e, "Sorry, I don't have any recent messages from %s to work with.", nick)
		return
	}

	var msgs []string
	for _, m := range user.RecentMessages {
		msgs = append(msgs, m.Message)
	}
	prompt := fmt.Sprintf("Target: %s\nRequested by: %s\n\nRecent messages:\n%s", nick, e.From, strings.Join(msgs, "\n"))

	task := models.NewProxyLLMRequestTask(channel, e.From, "roast", prompt)
	if err := queue.GetProxy().Publish(task); err != nil {
		logger.Errorf(e, "error publishing roast request: %s", err)
		return
	}
}
