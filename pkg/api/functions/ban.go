package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const banFunctionName = "ban"

type banFunction struct {
	FunctionStub
}

func NewBanFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, banFunctionName)
	if err != nil {
		return nil, err
	}

	return &banFunction{
		FunctionStub: stub,
	}, nil
}

func (f *banFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *banFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	user := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] ban %s %s", e.From, e.ReplyTarget(), channel, user)

	f.isBotAuthorizedByChannelStatus(channel, irc.HalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			f.Replyf(e, "Missing required permissions to kick users in this channel. Did you forget /mode %s +h %s?", channel, f.cfg.Connection.Nick)
			return
		}

		reason := ""
		if len(tokens) > 2 {
			reason = strings.Join(tokens[2:], " ")
		}
		f.irc.Ban(channel, user, reason)
		logger.Infof(e, "banned %s in %s", user, channel)
	})
}
