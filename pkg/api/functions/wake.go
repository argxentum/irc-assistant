package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const wakeFunctionName = "wake"

type wakeFunction struct {
	*functionStub
}

func NewWakeFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &wakeFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (f *wakeFunction) Name() string {
	return wakeFunctionName
}

func (f *wakeFunction) Description() string {
	return "Wakes the bot, enabling it across all channels."
}

func (f *wakeFunction) Triggers() []string {
	return []string{"wake"}
}

func (f *wakeFunction) Usages() []string {
	return []string{"%s"}
}

func (f *wakeFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *wakeFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 0)
}

func (f *wakeFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", f.Name(), e.From, e.ReplyTarget())

	if f.ctx.Session().IsAwake {
		logger.Warningf(e, "already awake")
		f.Replyf(e, "Already awake.")
		return
	}

	f.ctx.Session().IsAwake = true
	logger.Debug(e, "awake")
	f.SendMessage(e, e.ReplyTarget(), "Now awake.")
}
