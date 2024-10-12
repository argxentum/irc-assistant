package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const tempBanFunctionName = "tempban"

type tempBanFunction struct {
	FunctionStub
}

func NewTempBanFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, tempBanFunctionName)
	if err != nil {
		return nil, err
	}

	return &tempBanFunction{
		FunctionStub: stub,
	}, nil
}

func (f *tempBanFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 3)
}

func (f *tempBanFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()
	nick := tokens[1]

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] tempban %s %s", e.From, e.ReplyTarget(), channel, nick)

	f.isBotAuthorizedByChannelStatus(channel, irc.HalfOperator, func(authorized bool) {
		if !authorized {
			f.Replyf(e, "Missing required permissions to temporarily ban users in this channel. Did you forget /mode %s +h %s?", channel, f.cfg.Client.Nick)
			return
		}

		reason := ""
		if len(tokens) > 3 {
			reason = strings.Join(tokens[3:], " ")
		}

		f.irc.TemporaryBan(channel, nick, reason, 0)
		logger.Infof(e, "temporarily banned %s from %s", nick, channel)
	})
}
