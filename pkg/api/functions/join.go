package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"strings"
	"time"
)

const joinFunctionName = "join"

type joinFunction struct {
	stub
}

func NewJoinFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, joinFunctionName)
	if err != nil {
		return nil, err
	}

	return &joinFunction{
		stub: stub,
	}, nil
}

func (f *joinFunction) ShouldExecute(e *core.Event) bool {
	ok, tokens := f.verifyInput(e, 1)
	return ok && (strings.HasPrefix(tokens[1], "#") || strings.HasPrefix(tokens[1], "&"))
}

func (f *joinFunction) Execute(e *core.Event) error {
	tokens := parseTokens(e.Message())

	for _, token := range tokens[1:] {
		if !strings.HasPrefix(token, "#") && !strings.HasPrefix(token, "&") {
			continue
		}

		go func() {
			f.irc.Join(tokens[1])
			time.Sleep(250 * time.Millisecond)
		}()
	}

	return nil
}
