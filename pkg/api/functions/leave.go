package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"strings"
	"time"
)

const leaveFunctionName = "leave"

type leaveFunction struct {
	stub
}

func NewLeaveFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, leaveFunctionName)
	if err != nil {
		return nil, err
	}

	return &leaveFunction{
		stub: stub,
	}, nil
}

func (f *leaveFunction) ShouldExecute(e *core.Event) bool {
	ok, tokens := f.verifyInput(e, 0)
	if len(tokens) == 1 && !e.IsPrivateMessage() {
		return ok
	}

	return ok && (strings.HasPrefix(tokens[1], "#") || strings.HasPrefix(tokens[1], "&"))
}

func (f *leaveFunction) Execute(e *core.Event) error {
	tokens := parseTokens(e.Message())

	if len(tokens) == 1 && !e.IsPrivateMessage() {
		f.irc.Part(e.ReplyTarget())
		return nil
	}

	for _, token := range tokens[1:] {
		if !strings.HasPrefix(token, "#") && !strings.HasPrefix(token, "&") {
			continue
		}

		go func() {
			f.irc.Part(tokens[1])
			time.Sleep(250 * time.Millisecond)
		}()
	}

	return nil
}
