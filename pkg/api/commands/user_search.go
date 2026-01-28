package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const UserSearchCommandName = "user_search"

type UserSearchCommand struct {
	*commandStub
}

func NewUserSearchCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &UserSearchCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *UserSearchCommand) Name() string {
	return UserSearchCommandName
}

func (c *UserSearchCommand) Description() string {
	return "Searches users by mask."
}

func (c *UserSearchCommand) Triggers() []string {
	return []string{"us", "usersearch"}
}

func (c *UserSearchCommand) Usages() []string {
	return []string{"%s [<channel>] <mask>"}
}

func (c *UserSearchCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *UserSearchCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *UserSearchCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if len(tokens) > 2 && irc.IsChannel(tokens[1]) {
		channel = tokens[1]
		tokens = append(tokens[:1], tokens[2:]...)
	}

	mask := tokens[1]

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), channel)

	messages := make([]string, 0)
	c.authorizer.ListUsersByMask(channel, mask, func(users []*irc.User) {
		nicks := make([]string, 0)
		for _, user := range users {
			nicks = append(nicks, user.Mask.Nick)
		}

		plural := "s"
		if len(users) == 1 {
			plural = ""
		}

		messages = append(messages, fmt.Sprintf("Found %s user%s matching mask %s in channel %s: %s", style.Bold(fmt.Sprintf("%d", len(users))), plural, style.Bold(mask), channel, strings.Join(nicks, ", ")))
	})

}
