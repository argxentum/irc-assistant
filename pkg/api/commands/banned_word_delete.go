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

const bannedWordDeleteCommandName = "bannedWordDelete"

type bannedWordDeleteCommand struct {
	*commandStub
}

func NewBannedWordDeleteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &bannedWordDeleteCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *bannedWordDeleteCommand) Name() string {
	return bannedWordDeleteCommandName
}

func (c *bannedWordDeleteCommand) Description() string {
	return "Removes a word from the channel's banned words list."
}

func (c *bannedWordDeleteCommand) Triggers() []string {
	return []string{"bwdel"}
}

func (c *bannedWordDeleteCommand) Usages() []string {
	return []string{
		"%s <word> (in a channel)",
		"%s <channel> <word1> [<word2> ...] (outside a channel)",
	}
}

func (c *bannedWordDeleteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *bannedWordDeleteCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		c.Authorizer().IsAuthorized(e, tokens[1], callback)
	} else {
		c.Authorizer().IsAuthorized(e, channel, callback)
	}
}
func (c *bannedWordDeleteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *bannedWordDeleteCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) < 3 {
		c.Replyf(e, "Invalid usage. See %s for more information.", style.Italics(fmt.Sprintf("%s%s %s", c.cfg.Commands.Prefix, registry.Command(helpCommandName).Triggers()[0], strings.TrimPrefix(tokens[0], c.cfg.Commands.Prefix))))
		return
	}

	channel := e.ReplyTarget()
	words := make([]string, 0)

	if len(tokens) > 2 {
		channel = tokens[1]
		words = tokens[2:]
	} else {
		words = tokens[1:]
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, strings.Join(words, ", "))

	store := firestore.Get()
	for _, word := range words {
		err := store.RemoveBannedWord(channel, word)
		if err != nil {
			logger.Errorf(e, "error removing banned word: %s", err)
			return
		}
	}

	for _, word := range words {
		c.ctx.Session().RemoveBannedWord(channel, word)
	}

	c.Replyf(e, "Updated banned words in %s.", style.Bold(channel))
}
