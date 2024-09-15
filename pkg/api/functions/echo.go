package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"strings"
)

const echoFunctionName = "echo"

type echoFunction struct {
	stub
}

func NewEchoFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, echoFunctionName)
	if err != nil {
		return nil, err
	}

	return &echoFunction{
		stub: stub,
	}, nil
}

func (f *echoFunction) Matches(e *core.Event) bool {
	if !f.isAuthorized(e) {
		return false
	}

	tokens := sanitizedTokens(e.Message(), 200)
	if len(tokens) < 2 {
		return false
	}
	return tokens[0] == f.Prefix
}

func (f *echoFunction) Execute(e *core.Event) error {
	tokens := sanitizedTokens(e.Message(), 200)
	f.irc.SendMessage(e.ReplyTarget(), strings.Join(tokens[1:], " "))
	return nil
}
