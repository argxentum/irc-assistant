package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
	"slices"
	"strings"
)

const DisinfoWarningDeleteCommandName = "delete_disinfo_warning"

type DisinfoWarningDeleteCommand struct {
	*commandStub
}

func NewDisinfoWarningDeleteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &DisinfoWarningDeleteCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *DisinfoWarningDeleteCommand) Name() string {
	return DisinfoWarningDeleteCommandName
}

func (c *DisinfoWarningDeleteCommand) Description() string {
	return "Removes a URL prefix from the channel's disinformation warning list."
}

func (c *DisinfoWarningDeleteCommand) Triggers() []string {
	return []string{"dwdel"}
}

func (c *DisinfoWarningDeleteCommand) Usages() []string {
	return []string{
		"%s <word> (in a channel)",
		"%s <channel> <word1> [<word2> ...] (outside a channel)",
	}
}

func (c *DisinfoWarningDeleteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *DisinfoWarningDeleteCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		c.Authorizer().IsAuthorized(e, tokens[1], callback)
	} else {
		c.Authorizer().IsAuthorized(e, channel, callback)
	}
}
func (c *DisinfoWarningDeleteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *DisinfoWarningDeleteCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	store := firestore.Get()
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) < 3 {
		c.Replyf(e, "Invalid usage. See %s for more information.", style.Italics(fmt.Sprintf("%s%s %s", c.cfg.Commands.Prefix, registry.Command(HelpCommandName).Triggers()[0], strings.TrimPrefix(tokens[0], c.cfg.Commands.Prefix))))
		return
	}

	channelName := e.ReplyTarget()
	if e.IsPrivateMessage() {
		channelName = tokens[1]
	}

	channel, err := repository.GetChannel(e, channelName)
	if err != nil {
		logger.Errorf(e, "error retrieving channel: %s", err)
		return
	}

	var urlPrefixes []string
	if e.IsPrivateMessage() {
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
