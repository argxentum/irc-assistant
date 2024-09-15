package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"strings"
)

const kickFunctionName = "kick"

type kickFunction struct {
	stub
}

func NewKickFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, kickFunctionName)
	if err != nil {
		return nil, err
	}

	return &kickFunction{
		stub: stub,
	}, nil
}

func (f *kickFunction) ShouldExecute(e *core.Event) bool {
	if e.IsPrivateMessage() {
		return false
	}

	tokens := parseTokens(e.Message())
	if len(tokens) < 2 {
		return false
	}

	for _, t := range f.Triggers {
		if strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix) == t {
			return true
		}
	}
	return false
}

func (f *kickFunction) Execute(e *core.Event) error {
	sender, _ := e.Sender()
	f.irc.GetUserStatus(e.ReplyTarget(), sender, func(status string) {
		if !f.isSenderAuthorized(sender, f.Authorization) && status != core.Operator && status != core.HalfOperator {
			return
		}

		tokens := parseTokens(e.Message())
		channel := e.ReplyTarget()
		user := tokens[1]
		reason := ""
		if len(tokens) > 2 {
			reason = strings.Join(tokens[2:], " ")
		}
		f.irc.Kick(channel, user, reason)
	})

	return nil
}
