package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
	"strings"
)

const banFunctionName = "ban"

type banFunction struct {
	FunctionStub
}

func NewBanFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, banFunctionName)
	if err != nil {
		return nil, err
	}

	return &banFunction{
		FunctionStub: stub,
	}, nil
}

func (f *banFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *banFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ ban\n")
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
		f.irc.Ban(channel, user, reason)
	})

}
