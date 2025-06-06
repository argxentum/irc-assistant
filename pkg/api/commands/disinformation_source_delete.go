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

const DisinformationSourceDeleteCommandName = "delete_disinformation_source"

type DisinformationSourceDeleteCommand struct {
	*commandStub
}

func NewDisinformationSourceDeleteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &DisinformationSourceDeleteCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *DisinformationSourceDeleteCommand) Name() string {
	return DisinformationSourceDeleteCommandName
}

func (c *DisinformationSourceDeleteCommand) Description() string {
	return "Deletes a disinformation source for the channel."
}

func (c *DisinformationSourceDeleteCommand) Triggers() []string {
	return []string{"disinfodel"}
}

func (c *DisinformationSourceDeleteCommand) Usages() []string {
	return []string{
		"%s <word> (in a channel)",
		"%s <channel> <word1> [<word2> ...] (outside a channel)",
	}
}

func (c *DisinformationSourceDeleteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *DisinformationSourceDeleteCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		c.Authorizer().IsAuthorized(e, tokens[1], callback)
	} else {
		c.Authorizer().IsAuthorized(e, channel, callback)
	}
}
func (c *DisinformationSourceDeleteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *DisinformationSourceDeleteCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	fs := firestore.Get()
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
		if err := fs.DeleteDisinformationSource(channelName, urlPrefix); err != nil {
			logger.Errorf(e, "error deleting disinformation source: %v", err)
		}
	}

	c.Replyf(e, "Updated disinformation sources in %s.", style.Bold(channelName))
}
