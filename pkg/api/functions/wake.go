package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
)

const wakeFunctionName = "wake"

type wakeFunction struct {
	Stub
}

func NewWakeFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, wakeFunctionName)
	if err != nil {
		return nil, err
	}

	return &wakeFunction{
		Stub: stub,
	}, nil
}

func (f *wakeFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 0)
}

func (f *wakeFunction) Execute(e *core.Event) {
	fmt.Printf("Executing function: wake\n")

	if f.isAwake() {
		f.Reply(e, "Already awake.")
		return
	}

	f.ctx.Set(context.IsAwakeKey, true)
	f.irc.SendMessage(e.ReplyTarget(), "Now awake.")
}
