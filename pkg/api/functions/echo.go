package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
	"strings"
)

const echoFunctionName = "echo"

type echoFunction struct {
	FunctionStub
}

func NewEchoFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, echoFunctionName)
	if err != nil {
		return nil, err
	}

	return &echoFunction{
		FunctionStub: stub,
	}, nil
}

func (f *echoFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *echoFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ echo\n")
	tokens := Tokens(e.Message())
	f.irc.SendMessage(e.ReplyTarget(), strings.Join(tokens[1:], " "))
}
