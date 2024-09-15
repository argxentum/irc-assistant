package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"strings"
)

const sayFunctionName = "say"

type sayFunction struct {
	stub
}

func NewSayFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, sayFunctionName)
	if err != nil {
		return nil, err
	}

	return &sayFunction{
		stub: stub,
	}, nil
}

func (f *sayFunction) ShouldExecute(e *core.Event) bool {
	ok, tokens := f.verifyInput(e, 3)
	return ok && (strings.HasPrefix(tokens[2], "#") || strings.HasPrefix(tokens[2], "&"))
}

func (f *sayFunction) Execute(e *core.Event) error {
	tokens := parseTokens(e.Message())
	f.irc.SendMessage(tokens[1], strings.Join(tokens[2:], " "))
	return nil
}
