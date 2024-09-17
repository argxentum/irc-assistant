package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"time"
)

const joinFunctionName = "join"

type joinFunction struct {
	Stub
}

func NewJoinFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, joinFunctionName)
	if err != nil {
		return nil, err
	}

	return &joinFunction{
		Stub: stub,
	}, nil
}

func (f *joinFunction) MayExecute(e *core.Event) bool {
	if !e.IsPrivateMessage() || !f.isValid(e, 1) {
		return false
	}

	tokens := Tokens(e.Message())
	return core.IsChannel(tokens[1])
}

func (f *joinFunction) Execute(e *core.Event) {
	tokens := Tokens(e.Message())

	for _, token := range tokens[1:] {
		if !core.IsChannel(token) {
			continue
		}

		go func() {
			f.irc.Join(tokens[1])
			time.Sleep(250 * time.Millisecond)
		}()
	}
}
