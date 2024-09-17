package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
	"strings"
)

const sayFunctionName = "say"

type sayFunction struct {
	Stub
}

func NewSayFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, sayFunctionName)
	if err != nil {
		return nil, err
	}

	return &sayFunction{
		Stub: stub,
	}, nil
}

func (f *sayFunction) MayExecute(e *core.Event) bool {
	fmt.Printf("Executing function: say\n")
	if !f.isValid(e, 3) {
		return false
	}

	tokens := Tokens(e.Message())
	return core.IsChannel(tokens[1])
}

func (f *sayFunction) Execute(e *core.Event) {
	tokens := Tokens(e.Message())
	f.irc.SendMessage(tokens[1], strings.Join(tokens[2:], " "))
}
