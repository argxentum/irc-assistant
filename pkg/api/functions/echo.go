package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const echoFunctionName = "echo"

type echoFunction struct {
	FunctionStub
}

func NewEchoFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, echoFunctionName)
	if err != nil {
		return nil, err
	}

	return &echoFunction{
		FunctionStub: stub,
	}, nil
}

func (f *echoFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *echoFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], " ")
	log.Logger().Infof(e, "⚡ [%s/%s] echo %s", e.From, e.ReplyTarget(), message)
	f.SendMessage(e, e.ReplyTarget(), message)
}
