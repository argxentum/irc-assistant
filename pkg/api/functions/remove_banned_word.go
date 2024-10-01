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

const removeBannedWordFunctionName = "removeBannedWord"

type removeBannedWordFunction struct {
	FunctionStub
}

func NewRemoveBannedWordFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, removeBannedWordFunctionName)
	if err != nil {
		return nil, err
	}

	return &removeBannedWordFunction{
		FunctionStub: stub,
	}, nil
}

func (f *removeBannedWordFunction) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		f.FunctionStub.IsAuthorized(e, tokens[1], callback)
	} else {
		f.FunctionStub.IsAuthorized(e, channel, callback)
	}
}
func (f *removeBannedWordFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *removeBannedWordFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) < 3 {
		f.Replyf(e, "Invalid usage. See %s for more information.", style.Italics(fmt.Sprintf("%s%s %s", f.cfg.Functions.Prefix, f.functionConfig(helpFunctionName).Triggers[0], strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix))))
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
	logger.Infof(e, "⚡ [%s/%s] removeBannedWord %s %s", e.From, e.ReplyTarget(), channel, strings.Join(words, ", "))

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
