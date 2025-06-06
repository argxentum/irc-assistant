package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const DisinformationSourceAddCommandName = "add_disinformation_source"

type DisinformationSourceAddCommand struct {
	*commandStub
}

func NewDisinformationSourceAddCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &DisinformationSourceAddCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *DisinformationSourceAddCommand) Name() string {
	return DisinformationSourceAddCommandName
}

func (c *DisinformationSourceAddCommand) Description() string {
	return "Adds a disinformation source for the channel. Any URL that begins with the source will be considered possible disinformation."
}

func (c *DisinformationSourceAddCommand) Triggers() []string {
	return []string{"disinfoadd"}
}

func (c *DisinformationSourceAddCommand) Usages() []string {
	return []string{
		"%s <word> (in a channel)",
		"%s <channel> <word1> [<word2> ...] (outside a channel)",
	}
}

func (c *DisinformationSourceAddCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *DisinformationSourceAddCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		c.commandStub.authorizer.IsAuthorized(e, tokens[1], callback)
	} else {
		c.commandStub.authorizer.IsAuthorized(e, channel, callback)
	}
}

func (c *DisinformationSourceAddCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *DisinformationSourceAddCommand) Execute(e *irc.Event) {
	fs := firestore.Get()
	logger := log.Logger()
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) < 3 {
		c.Replyf(e, "Invalid usage. See %s for more information.", style.Italics(fmt.Sprintf("%s%s %s", c.cfg.Commands.Prefix, registry.Command(HelpCommandName).Triggers()[0], strings.TrimPrefix(tokens[0], c.cfg.Commands.Prefix))))
		return
	}

	channelName := e.ReplyTarget()
	if e.IsPrivateMessage() {
		channelName = tokens[1]
	}

	var urlPrefixes []string
	if e.IsPrivateMessage() {
		urlPrefixes = tokens[2:]
	} else {
		urlPrefixes = tokens[1:]
	}

	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channelName, strings.Join(urlPrefixes, ", "))

	for _, urlPrefix := range urlPrefixes {
		if err := fs.AddDisinformationSource(channelName, urlPrefix); err != nil {
			logger.Errorf(e, "error adding disinformation source: %v", err)
		}
	}

	c.Replyf(e, "Updated disinformation sources in %s.", style.Bold(channelName))
}
