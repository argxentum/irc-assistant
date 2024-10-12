package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
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
	mask := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] ban %s %s", e.From, e.ReplyTarget(), channel, mask)

	f.isBotAuthorizedByChannelStatus(channel, irc.HalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			f.Replyf(e, "Missing required permissions to ban users in this channel. Did you forget /mode %s +h %s?", channel, f.cfg.IRC.Nick)
			return
		}

		f.irc.Ban(channel, mask)
		logger.Infof(e, "banned %s in %s", mask, channel)
	})
}
