package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const aboutFunctionName = "about"

type aboutFunction struct {
	*functionStub
}

func NewAboutFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &aboutFunction{
		functionStub: defaultFunctionStub(ctx, cfg, ircs),
	}
}

func (f *aboutFunction) Name() string {
	return aboutFunctionName
}

func (f *aboutFunction) Description() string {
	return "Shows information about the bot."
}

func (f *aboutFunction) Triggers() []string {
	return []string{"about"}
}

func (f *aboutFunction) Usages() []string {
	return []string{"%s"}
}

func (f *aboutFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *aboutFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 0)
}

func (f *aboutFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", f.Name(), e.From, e.ReplyTarget())
	message := "Version 0.1. Source: https://github.com/argxentum/irc-assistant."
	f.SendMessage(e, e.ReplyTarget(), message)
}
