package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const sayFunctionName = "say"

type sayFunction struct {
	FunctionStub
}

func NewSayFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, sayFunctionName)
	if err != nil {
		return nil, err
	}

	return &sayFunction{
		FunctionStub: stub,
	}, nil
}

func (f *sayFunction) MayExecute(e *irc.Event) bool {
	if !f.isValid(e, 3) {
		return false
	}

	tokens := Tokens(e.Message())
	return irc.IsChannel(tokens[1])
}

func (f *sayFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := tokens[1]
	message := strings.Join(tokens[2:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] say %s %s", e.From, e.ReplyTarget(), channel, message)

	f.SendMessage(e, channel, message)
}
