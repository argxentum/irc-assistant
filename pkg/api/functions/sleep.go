package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
)

const sleepFunctionName = "sleep"

type sleepFunction struct {
	FunctionStub
}

func NewSleepFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, sleepFunctionName)
	if err != nil {
		return nil, err
	}

	return &sleepFunction{
		FunctionStub: stub,
	}, nil
}

func (f *sleepFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 0)
}

func (f *sleepFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ sleep\n")
	wakeTrigger := ""
	for k, v := range f.cfg.Functions.EnabledFunctions {
		if k == wakeFunctionName {
			if len(v.Triggers) > 0 {
				wakeTrigger = fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, v.Triggers[0])
			}
		}
	}
	f.ctx.SetAwake(false)
	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("Sleeping until awoken with %s.", text.Italics(wakeTrigger)))
}
