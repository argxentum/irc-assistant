package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"time"
)

const leaveFunctionName = "leave"

type leaveFunction struct {
	Stub
}

func NewLeaveFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, leaveFunctionName)
	if err != nil {
		return nil, err
	}

	return &leaveFunction{
		Stub: stub,
	}, nil
}

func (f *leaveFunction) MayExecute(e *core.Event) bool {
	tokens := Tokens(e.Message())
	if e.IsPrivateMessage() {
		return f.isValid(e, 1) && core.IsChannel(tokens[1])
	}

	return f.isValid(e, 0)
}

func (f *leaveFunction) Execute(e *core.Event) {
	tokens := Tokens(e.Message())

	if len(tokens) == 1 && !e.IsPrivateMessage() {
		f.irc.Part(e.ReplyTarget())
		return
	}

	for _, token := range tokens[1:] {
		if !core.IsChannel(token) {
			continue
		}

		go func() {
			f.irc.Part(tokens[1])
			time.Sleep(250 * time.Millisecond)
		}()
	}
}
