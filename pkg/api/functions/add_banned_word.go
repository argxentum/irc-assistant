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

const addBannedWordFunctionName = "addBannedWord"

type addBannedWordFunction struct {
	FunctionStub
}

func NewAddBannedWordFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, addBannedWordFunctionName)
	if err != nil {
		return nil, err
	}

	return &addBannedWordFunction{
		FunctionStub: stub,
	}, nil
}

func (f *addBannedWordFunction) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())

	if e.IsPrivateMessage() && len(tokens) > 2 {
		f.FunctionStub.IsAuthorized(e, tokens[1], callback)
	} else {
		f.FunctionStub.IsAuthorized(e, channel, callback)
	}
}

func (f *addBannedWordFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *addBannedWordFunction) Execute(e *irc.Event) {
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
	logger.Infof(e, "⚡ [%s/%s] addBannedWord %s %s", e.From, e.ReplyTarget(), channel, strings.Join(words, ", "))

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
