package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const aboutFunctionName = "about"

type aboutFunction struct {
	FunctionStub
}

func NewAboutFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, aboutFunctionName)
	if err != nil {
		return nil, err
	}

	return &aboutFunction{
		FunctionStub: stub,
	}, nil
}

func (f *aboutFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 0)
}

func (f *aboutFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] about", e.From, e.ReplyTarget())
	message := "Version 0.1. Source: https://github.com/argxentum/irc-assistant."
	f.SendMessage(e, e.ReplyTarget(), message)
}
