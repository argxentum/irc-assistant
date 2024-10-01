package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
)

const sleepFunctionName = "sleep"

type sleepFunction struct {
	FunctionStub
}

func NewSleepFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, sleepFunctionName)
	if err != nil {
		return nil, err
	}

	return &sleepFunction{
		FunctionStub: stub,
	}, nil
}

func (f *sleepFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 0)
}

func (f *sleepFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] sleep", e.From, e.ReplyTarget())

	wakeTrigger := ""
	for k, v := range f.cfg.Functions.EnabledFunctions {
		if k == wakeFunctionName {
			if len(v.Triggers) > 0 {
				wakeTrigger = fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, v.Triggers[0])
			}
		}
	}

	f.ctx.Session().IsAwake = false
	logger.Debug(e, "sleeping")
	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("Sleeping until awoken with %s.", style.Italics(wakeTrigger)))
}
