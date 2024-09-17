package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"strings"
)

const banFunctionName = "ban"

type banFunction struct {
	Stub
}

func NewBanFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, banFunctionName)
	if err != nil {
		return nil, err
	}

	return &banFunction{
		Stub: stub,
	}, nil
}

func (f *banFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *banFunction) Execute(e *core.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()
	user := tokens[1]
	reason := ""
	if len(tokens) > 2 {
		reason = strings.Join(tokens[2:], " ")
	}
	f.irc.Ban(channel, user, reason)
}
