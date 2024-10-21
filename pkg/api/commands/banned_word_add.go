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

const bannedWordAddCommandName = "bannedWordAdd"

type bannedWordAddCommand struct {
	*commandStub
}

func NewBannedWordAddCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &bannedWordAddCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *bannedWordAddCommand) Name() string {
	return bannedWordAddCommandName
}

func (c *bannedWordAddCommand) Description() string {
	return "Adds a word to the channel's banned words list."
}

func (c *bannedWordAddCommand) Triggers() []string {
	return []string{"bwadd"}
}

func (c *bannedWordAddCommand) Usages() []string {
	return []string{
		"%s <word> (in a channel)",
		"%s <channel> <word1> [<word2> ...] (outside a channel)",
	}
}

func (c *bannedWordAddCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *bannedWordAddCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		c.commandStub.authorizer.IsAuthorized(e, tokens[1], callback)
	} else {
		c.commandStub.authorizer.IsAuthorized(e, channel, callback)
	}
}

func (c *bannedWordAddCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *bannedWordAddCommand) Execute(e *irc.Event) {
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
		err := store.AddBannedWord(c.ctx, channel, word)
		if err != nil {
			logger.Errorf(e, "error adding banned word: %s", err)
			return
		}
	}

	for _, word := range words {
		c.ctx.Session().AddBannedWord(channel, word)
	}

	c.Replyf(e, "Updated banned words in %s.", style.Bold(channel))
}
