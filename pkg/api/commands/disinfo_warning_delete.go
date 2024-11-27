package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
	"slices"
	"strings"
)

const disinfoWarningDeleteCommandName = "disinfoWarningDelete"

type disinfoWarningDeleteCommand struct {
	*commandStub
}

func NewDisinfoWarningDeleteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &disinfoWarningDeleteCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *disinfoWarningDeleteCommand) Name() string {
	return disinfoWarningDeleteCommandName
}

func (c *disinfoWarningDeleteCommand) Description() string {
	return "Removes a URL prefix from the channel's disinformation warning list."
}

func (c *disinfoWarningDeleteCommand) Triggers() []string {
	return []string{"dwdel"}
}

func (c *disinfoWarningDeleteCommand) Usages() []string {
	return []string{
		"%s <word> (in a channel)",
		"%s <channel> <word1> [<word2> ...] (outside a channel)",
	}
}

func (c *disinfoWarningDeleteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *disinfoWarningDeleteCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		c.Authorizer().IsAuthorized(e, tokens[1], callback)
	} else {
		c.Authorizer().IsAuthorized(e, channel, callback)
	}
}
func (c *disinfoWarningDeleteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *disinfoWarningDeleteCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	store := firestore.Get()
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) < 3 {
		c.Replyf(e, "Invalid usage. See %s for more information.", style.Italics(fmt.Sprintf("%s%s %s", c.cfg.Commands.Prefix, registry.Command(helpCommandName).Triggers()[0], strings.TrimPrefix(tokens[0], c.cfg.Commands.Prefix))))
		return
	}

	channelName := e.ReplyTarget()
	channel, err := store.Channel(channelName)
	if err != nil {
		logger.Errorf(e, "error retrieving channel: %s", err)
		return
	}

	var urlPrefixes []string
	if len(tokens) > 2 {
		channelName = tokens[1]
		urlPrefixes = tokens[2:]
	} else {
		urlPrefixes = tokens[1:]
	}

	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channelName, strings.Join(urlPrefixes, ", "))

	warnings := make([]string, 0)
	for _, warning := range channel.Summarization.DisinformationWarnings {
		if !slices.Contains(urlPrefixes, warning) {
			warnings = append(warnings, warning)
		}
	}

	channel.Summarization.DisinformationWarnings = warnings

	if err := store.UpdateChannel(channelName, map[string]interface{}{"summarization": channel.Summarization}); err != nil {
		logger.Errorf(e, "error updating channel summarization: %s", err)
		return
	}

	c.Replyf(e, "Updated disinformation warnings in %s.", style.Bold(channelName))
}
