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
	*functionStub
}

func NewSleepFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &sleepFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (f *sleepFunction) Name() string {
	return sleepFunctionName
}

func (f *sleepFunction) Description() string {
	return "Puts the bot to sleep, disabling it across all channels until awakened."
}

func (f *sleepFunction) Triggers() []string {
	return []string{"sleep"}
}

func (f *sleepFunction) Usages() []string {
	return []string{"%s"}
}

func (f *sleepFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *sleepFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 0)
}

func (f *sleepFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", f.Name(), e.From, e.ReplyTarget())

	wakeTrigger := ""
	for k, v := range registry.Functions() {
		if k == wakeFunctionName {
			if len(v.Triggers()) > 0 {
				wakeTrigger = fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, v.Triggers()[0])
			}
		}
	}

	f.ctx.Session().IsAwake = false
	logger.Debug(e, "sleeping")
	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("Sleeping until awoken with %s.", style.Italics(wakeTrigger)))
}
