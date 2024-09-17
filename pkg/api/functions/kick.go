package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
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
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()
	f.isBotAuthorizedByChannelStatus(channel, core.HalfOperator, func(authorized bool) {
		if !authorized {
			f.Reply(e, "Missing required permissions to kick users in this channel. Did you forget /mode %s +h %s?", channel, f.cfg.Connection.Nick)
			return
		}

		user := tokens[1]
		reason := ""
		if len(tokens) > 2 {
			reason = strings.Join(tokens[2:], " ")
		}
		f.irc.Kick(channel, user, reason)
	})
}
