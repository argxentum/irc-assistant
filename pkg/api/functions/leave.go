package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const leaveFunctionName = "leave"

type leaveFunction struct {
	*functionStub
}

func NewLeaveFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &leaveFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (f *leaveFunction) Name() string {
	return leaveFunctionName
}

func (f *leaveFunction) Description() string {
	return "Leaves the specified channel(s)."
}

func (f *leaveFunction) Triggers() []string {
	return []string{"leave", "part"}
}

func (f *leaveFunction) Usages() []string {
	return []string{"%s [<channel1> [<channel2> ...]]"}
}

func (f *leaveFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *leaveFunction) CanExecute(e *irc.Event) bool {
	tokens := Tokens(e.Message())
	if e.IsPrivateMessage() {
		return f.isFunctionEventValid(f, e, 1) && irc.IsChannel(tokens[1])
	}

	return f.isFunctionEventValid(f, e, 0)
}

func (f *leaveFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channels := tokens[1:]

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), strings.Join(channels, ", "))

	if len(tokens) == 1 && !e.IsPrivateMessage() {
		f.irc.Part(e.ReplyTarget())
		logger.Infof(e, "left %s", e.ReplyTarget())
		return
	}

	for _, channel := range channels {
		if !irc.IsChannel(channel) {
			continue
		}

		f.irc.Part(channel)
		logger.Infof(e, "left %s", channel)
	}
}
