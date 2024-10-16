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

const bannedWordDeleteFunctionName = "bannedWordDelete"

type bannedWordDeleteFunction struct {
	*functionStub
}

func NewBannedWordDeleteFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &bannedWordDeleteFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (f *bannedWordDeleteFunction) Name() string {
	return bannedWordDeleteFunctionName
}

func (f *bannedWordDeleteFunction) Description() string {
	return "Removes a word from the channel's banned words list."
}

func (f *bannedWordDeleteFunction) Triggers() []string {
	return []string{"bwdel"}
}

func (f *bannedWordDeleteFunction) Usages() []string {
	return []string{
		"%s <word> (in a channel)",
		"%s <channel> <word1> [<word2> ...] (outside a channel)",
	}
}

func (f *bannedWordDeleteFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *bannedWordDeleteFunction) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		f.Authorizer().IsAuthorized(e, tokens[1], callback)
	} else {
		f.Authorizer().IsAuthorized(e, channel, callback)
	}
}
func (f *bannedWordDeleteFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *bannedWordDeleteFunction) Execute(e *irc.Event) {
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
		err := store.RemoveBannedWord(f.ctx, channel, word)
		if err != nil {
			logger.Errorf(e, "error removing banned word: %s", err)
			return
		}
	}

	for _, word := range words {
		f.ctx.Session().RemoveBannedWord(channel, word)
	}

	f.Replyf(e, "Updated banned words in %s.", style.Bold(channel))
}
