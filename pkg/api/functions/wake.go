package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const wakeFunctionName = "wake"

type wakeFunction struct {
	FunctionStub
}

func NewWakeFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, wakeFunctionName)
	if err != nil {
		return nil, err
	}

	return &wakeFunction{
		FunctionStub: stub,
	}, nil
}

func (f *wakeFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 0)
}

func (f *wakeFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] wake", e.From, e.ReplyTarget())

	if f.ctx.IsAwake() {
		logger.Warningf(e, "already awake")
		f.Replyf(e, "Already awake.")
		return
	}

	f.ctx.SetAwake(true)
	logger.Debug(e, "awake")
	f.SendMessage(e, e.ReplyTarget(), "Now awake.")
}
