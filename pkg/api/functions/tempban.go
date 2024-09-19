package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
	"strings"
)

const tempBanFunctionName = "tempban"

type tempBanFunction struct {
	FunctionStub
}

func NewTempBanFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, tempBanFunctionName)
	if err != nil {
		return nil, err
	}

	return &tempBanFunction{
		FunctionStub: stub,
	}, nil
}

func (f *tempBanFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 3)
}

func (f *tempBanFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ tempban\n")
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()
	f.isBotAuthorizedByChannelStatus(channel, core.HalfOperator, func(authorized bool) {
		if !authorized {
			f.Reply(e, "Missing required permissions to temporarily ban users in this channel. Did you forget /mode %s +h %s?", channel, f.cfg.Connection.Nick)
			return
		}

		user := tokens[1]
		reason := ""
		if len(tokens) > 3 {
			reason = strings.Join(tokens[3:], " ")
		}
		f.irc.TemporaryBan(channel, user, reason, 0)
	})
}
