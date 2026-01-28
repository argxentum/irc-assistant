package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const HostSearchCommandName = "host_search"

type HostSearchCommand struct {
	*commandStub
}

func NewHostSearchCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &HostSearchCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *HostSearchCommand) Name() string {
	return HostSearchCommandName
}

func (c *HostSearchCommand) Description() string {
	return "Searches for users matching the specified host."
}

func (c *HostSearchCommand) Triggers() []string {
	return []string{"hs", "hostsearch"}
}

func (c *HostSearchCommand) Usages() []string {
	return []string{"%s [<channel>] <host>"}
}

func (c *HostSearchCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *HostSearchCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *HostSearchCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if len(tokens) > 2 && irc.IsChannel(tokens[1]) {
		channel = tokens[1]
		tokens = append(tokens[:1], tokens[2:]...)
	}

	host := tokens[1]

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), channel)

	users, err := repository.GetUsersByHost(e, channel, host)
	if err != nil {
		logger.Errorf(e, "error retrieving users by host %s in channel %s: %s", host, channel, err)
		c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("Error while searching for users with host %s in channel %s", style.Bold(host), channel))
		return
	}
	if len(users) == 0 {
		c.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("No users found with host %s in channel %s.", style.Bold(host), channel))
		return
	}

	nicks := make([]string, 0)
	for _, user := range users {
		nicks = append(nicks, user.Nick)
	}

	plural := "s"
	if len(users) == 1 {
		plural = ""
	}

	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("Found %s user%s with host %s in channel %s: %s", style.Bold(fmt.Sprintf("%d", len(users))), plural, style.Bold(host), channel, strings.Join(nicks, ", ")))

}
