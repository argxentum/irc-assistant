package functions

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

const bannedWordAddFunctionName = "bannedWordAdd"

type bannedWordAddFunction struct {
	*functionStub
}

func NewBannedWordAddFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &bannedWordAddFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (f *bannedWordAddFunction) Name() string {
	return bannedWordAddFunctionName
}

func (f *bannedWordAddFunction) Description() string {
	return "Adds a word to the channel's banned words list."
}

func (f *bannedWordAddFunction) Triggers() []string {
	return []string{"bwadd"}
}

func (f *bannedWordAddFunction) Usages() []string {
	return []string{
		"%s <word> (in a channel)",
		"%s <channel> <word1> [<word2> ...] (outside a channel)",
	}
}

func (f *bannedWordAddFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *bannedWordAddFunction) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		f.functionStub.authorizer.IsAuthorized(e, tokens[1], callback)
	} else {
		f.functionStub.authorizer.IsAuthorized(e, channel, callback)
	}
}

func (f *bannedWordAddFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *bannedWordAddFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) < 3 {
		f.Replyf(e, "Invalid usage. See %s for more information.", style.Italics(fmt.Sprintf("%s%s %s", f.cfg.Functions.Prefix, registry.Function(helpFunctionName).Triggers()[0], strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix))))
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
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", f.Name(), e.From, e.ReplyTarget(), channel, strings.Join(words, ", "))

	store := firestore.Get()
	for _, word := range words {
		err := store.AddBannedWord(f.ctx, channel, word)
		if err != nil {
			logger.Errorf(e, "error adding banned word: %s", err)
			return
		}
	}

	for _, word := range words {
		f.ctx.Session().AddBannedWord(channel, word)
	}

	f.Replyf(e, "Updated banned words in %s.", style.Bold(channel))
}
