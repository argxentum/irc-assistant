package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const joinFunctionName = "join"

type joinFunction struct {
	FunctionStub
}

func NewJoinFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, joinFunctionName)
	if err != nil {
		return nil, err
	}

	return &joinFunction{
		FunctionStub: stub,
	}, nil
}

func (f *joinFunction) MayExecute(e *irc.Event) bool {
	if !e.IsPrivateMessage() || !f.isValid(e, 1) {
		return false
	}

	tokens := Tokens(e.Message())
	return irc.IsChannel(tokens[1])
}

func (f *joinFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channels := tokens[1:]

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] join %s", e.From, e.ReplyTarget(), strings.Join(channels, ", "))

	for _, channel := range channels {
		if !irc.IsChannel(channel) {
			continue
		}

		f.irc.Join(tokens[1])
		logger.Infof(e, "joined %s", channel)
	}
}
