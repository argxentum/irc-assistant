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
	FunctionStub
}

func NewLeaveFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, leaveFunctionName)
	if err != nil {
		return nil, err
	}

	return &leaveFunction{
		FunctionStub: stub,
	}, nil
}

func (f *leaveFunction) MayExecute(e *irc.Event) bool {
	tokens := Tokens(e.Message())
	if e.IsPrivateMessage() {
		return f.isValid(e, 1) && irc.IsChannel(tokens[1])
	}

	return f.isValid(e, 0)
}

func (f *leaveFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channels := tokens[1:]

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] leave %s", e.From, e.ReplyTarget(), strings.Join(channels, ", "))

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
