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

const DisinformationSourceCommandName = "disinformation_source"

const (
	disinfoActionAdd    = "add"
	disinfoActionRemove = "remove"
	disinfoActionVerify = "verify"
)

type DisinformationSourceCommand struct {
	*commandStub
}

func NewDisinformationSourceCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &DisinformationSourceCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *DisinformationSourceCommand) Name() string {
	return DisinformationSourceCommandName
}

func (c *DisinformationSourceCommand) Description() string {
	return "Adds, removes, or verifies a disinformation source for the channel. Any URL that begins with the source will be considered possible disinformation."
}

func (c *DisinformationSourceCommand) Triggers() []string {
	return []string{"disinfo", "dis"}
}

func (c *DisinformationSourceCommand) Usages() []string {
	return []string{
		"%s add/remove/verify <word> (in a channel)",
		"%s <channel> add/remove/verify <word1> [<word2> ...] (outside a channel)",
	}
}

func (c *DisinformationSourceCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *DisinformationSourceCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		c.commandStub.authorizer.IsAuthorized(e, tokens[1], callback)
	} else {
		c.commandStub.authorizer.IsAuthorized(e, channel, callback)
	}
}

func (c *DisinformationSourceCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 2)
}

func (c *DisinformationSourceCommand) Execute(e *irc.Event) {
	fs := firestore.Get()
	logger := log.Logger()
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) < 4 {
		c.Replyf(e, "Invalid usage. See %s for more information.", style.Italics(fmt.Sprintf("%s%s %s", c.cfg.Commands.Prefix, registry.Command(HelpCommandName).Triggers()[0], strings.TrimPrefix(tokens[0], c.cfg.Commands.Prefix))))
		return
	}

	action := strings.ToLower(tokens[1])
	channelName := e.ReplyTarget()
	if e.IsPrivateMessage() {
		action = strings.ToLower(tokens[2])
		channelName = tokens[1]
	}

	if action == "a" || action == "insert" {
		action = disinfoActionAdd
	} else if action == "delete" || action == "rm" || action == "rem" || action == "del" {
		action = disinfoActionRemove
	} else if action == "v" || action == "verify" || action == "check" || action == "confirm" {
		action = disinfoActionVerify
	}

	var sources []string
	if e.IsPrivateMessage() {
		sources = tokens[3:]
	} else {
		sources = tokens[2:]
	}

	logger.Infof(e, "⚡ %s [%s/%s] %s %s %s", c.Name(), e.From, e.ReplyTarget(), channelName, action, strings.Join(sources, ", "))

	for _, source := range sources {
		if action == disinfoActionAdd {
			if err := fs.AddDisinformationSource(channelName, source); err != nil {
				logger.Errorf(e, "error adding disinformation source: %v", err)
			}
			c.Replyf(e, "Updated disinformation sources in %s.", style.Bold(channelName))
		} else if action == disinfoActionRemove {
			if err := fs.DeleteDisinformationSource(channelName, source); err != nil {
				logger.Errorf(e, "error removing disinformation source: %v", err)
			}
			c.Replyf(e, "Updated disinformation sources in %s.", style.Bold(channelName))
		} else if action == disinfoActionVerify {
			if fs.IsDisinformationSource(channelName, source) {
				c.Replyf(e, "%s %s is identified as a possible source of disinformation in %s.", "⚠️", style.Bold(source), style.Bold(channelName))
			} else {
				c.Replyf(e, "%s %s is not identified as a possible source of disinformation in %s.", "🆗", style.Bold(source), style.Bold(channelName))
			}
		}
	}
}
