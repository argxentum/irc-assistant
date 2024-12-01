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

const disinfoWarningAddCommandName = "disinfoWarningAdd"

type disinfoWarningAddCommand struct {
	*commandStub
}

func NewDisinfoWarningAddCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &disinfoWarningAddCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *disinfoWarningAddCommand) Name() string {
	return disinfoWarningAddCommandName
}

func (c *disinfoWarningAddCommand) Description() string {
	return "Adds a URL prefix to the channel's disinformation warning list."
}

func (c *disinfoWarningAddCommand) Triggers() []string {
	return []string{"dwadd"}
}

func (c *disinfoWarningAddCommand) Usages() []string {
	return []string{
		"%s <word> (in a channel)",
		"%s <channel> <word1> [<word2> ...] (outside a channel)",
	}
}

func (c *disinfoWarningAddCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *disinfoWarningAddCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		c.commandStub.authorizer.IsAuthorized(e, tokens[1], callback)
	} else {
		c.commandStub.authorizer.IsAuthorized(e, channel, callback)
	}
}

func (c *disinfoWarningAddCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *disinfoWarningAddCommand) Execute(e *irc.Event) {
	store := firestore.Get()
	logger := log.Logger()
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) < 3 {
		c.Replyf(e, "Invalid usage. See %s for more information.", style.Italics(fmt.Sprintf("%s%s %s", c.cfg.Commands.Prefix, registry.Command(helpCommandName).Triggers()[0], strings.TrimPrefix(tokens[0], c.cfg.Commands.Prefix))))
		return
	}

	channelName := e.ReplyTarget()
	if e.IsPrivateMessage() {
		channelName = tokens[1]
	}

	channel, err := store.Channel(channelName)
	if err != nil {
		logger.Errorf(e, "error retrieving channel: %s", err)
		return
	}

	if channel == nil {
		c.Replyf(e, "Channel %s not found.", style.Bold(channelName))
		return
	}

	var urlPrefixes []string
	if e.IsPrivateMessage() {
		urlPrefixes = tokens[2:]
	} else {
		urlPrefixes = tokens[1:]
	}

	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channelName, strings.Join(urlPrefixes, ", "))

	newEntries := make([]string, 0)

	for _, urlPrefix := range urlPrefixes {
		if slices.Contains(channel.Summarization.DisinformationWarnings, urlPrefix) {
			continue
		}
		newEntries = append(newEntries, urlPrefix)
	}

	if len(newEntries) == 0 {
		return
	}

	channel.Summarization.DisinformationWarnings = append(channel.Summarization.DisinformationWarnings, newEntries...)
	if err := store.UpdateChannel(channelName, map[string]any{"summarization": channel.Summarization}); err != nil {
		logger.Errorf(e, "error updating channel summarization: %s", err)
		return
	}

	c.Replyf(e, "Updated disinformation warnings in %s.", style.Bold(channelName))
}
