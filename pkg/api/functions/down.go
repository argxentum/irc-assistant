package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
)

const downFunctionName = "down"

type downFunction struct {
	FunctionStub
}

func NewDownFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, downFunctionName)
	if err != nil {
		return nil, err
	}

	return &downFunction{
		FunctionStub: stub,
	}, nil
}

func (f *downFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 0)
}

func (f *downFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ down\n")
	f.irc.Down(e.ReplyTarget(), e.Source)
}
