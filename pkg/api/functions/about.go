package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
)

const aboutFunctionName = "about"

type aboutFunction struct {
	Stub
}

func NewAboutFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, aboutFunctionName)
	if err != nil {
		return nil, err
	}

	return &aboutFunction{
		Stub: stub,
	}, nil
}

func (f *aboutFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 0)
}

func (f *aboutFunction) Execute(e *core.Event) {
	fmt.Printf("Executing function: about\n")
	f.irc.SendMessage(e.ReplyTarget(), "Version 0.1. Source: https://github.com/argxentum/irc-assistant.")
}
