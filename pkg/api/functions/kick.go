package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"strings"
)

const kickFunctionName = "kick"

type kickFunction struct {
	Stub
}

func NewKickFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, kickFunctionName)
	if err != nil {
		return nil, err
	}

	return &kickFunction{
		Stub: stub,
	}, nil
}

func (f *kickFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *kickFunction) Execute(e *core.Event) {
	f.IsAuthorized(e, func(authorized bool) {
		tokens := Tokens(e.Message())

		if !authorized {
			f.Reply(e, "%s: you are not authorized to use %s.", e.From, text.Italics(tokens[0]))
			return
		}

		channel := e.ReplyTarget()
		user := tokens[1]
		reason := ""
		if len(tokens) > 2 {
			reason = strings.Join(tokens[2:], " ")
		}
		f.irc.Kick(channel, user, reason)
	})
}
