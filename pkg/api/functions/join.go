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

func (f *joinFunction) Matches(e *core.Event) bool {
	if !f.isAuthorized(e) {
		return false
	}

	tokens := sanitizedTokens(e.Message(), 200)
	if len(tokens) < 2 {
		return false
	}
	return tokens[0] == f.Prefix && (strings.HasPrefix(tokens[1], "#") || strings.HasPrefix(tokens[1], "&"))
}

func (f *joinFunction) Execute(e *core.Event) error {
	tokens := sanitizedTokens(e.Message(), 200)

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
