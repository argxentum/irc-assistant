package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
)

const upFunctionName = "up"

type upFunction struct {
	FunctionStub
}

func NewUpFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, upFunctionName)
	if err != nil {
		return nil, err
	}

	return &upFunction{
		FunctionStub: stub,
	}, nil
}

func (f *upFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 0)
}

func (f *upFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ up\n")
	f.irc.Up(e.ReplyTarget(), e.Source)
}
