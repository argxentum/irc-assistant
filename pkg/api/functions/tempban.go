package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"strings"
)

const tempBanFunctionName = "tempban"

type tempBanFunction struct {
	Stub
}

func NewTempBanFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, tempBanFunctionName)
	if err != nil {
		return nil, err
	}

	return &tempBanFunction{
		Stub: stub,
	}, nil
}

func (f *tempBanFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 3)
}

func (f *tempBanFunction) Execute(e *core.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()
	user := tokens[1]
	reason := ""
	if len(tokens) > 3 {
		reason = strings.Join(tokens[3:], " ")
	}
	f.irc.TemporaryBan(channel, user, reason, 0)
}
