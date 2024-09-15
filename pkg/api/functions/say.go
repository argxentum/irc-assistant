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

func (f *sayFunction) Matches(e *core.Event) bool {
	if !f.isAuthorized(e) {
		return false
	}

	tokens := sanitizedTokens(e.Message(), 200)
	if len(tokens) < 3 {
		return false
	}

	for _, p := range f.Prefixes {
		if tokens[0] == p {
			return true
		}
	}
	return false
}

func (f *sayFunction) Execute(e *core.Event) error {
	tokens := sanitizedTokens(e.Message(), 200)
	f.irc.SendMessage(tokens[1], strings.Join(tokens[2:], " "))
	return nil
}
