package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
	"time"
)

const kickBanFunctionName = "kickban"

type kickBanFunction struct {
	FunctionStub
}

func NewKickBanFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, kickBanFunctionName)
	if err != nil {
		return nil, err
	}

	return &kickBanFunction{
		FunctionStub: stub,
	}, nil
}

func (f *kickBanFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *kickBanFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()
	nick := tokens[1]
	reason := ""
	if len(tokens) > 2 {
		reason = strings.Join(tokens[2:], " ")
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] kickBan %s %s - %s", e.From, e.ReplyTarget(), channel, nick, reason)

	f.isBotAuthorizedByChannelStatus(channel, irc.HalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			f.Replyf(e, "Missing required permissions to kick users in this channel. Did you forget /mode %s +h %s?", channel, f.cfg.IRC.Nick)
			return
		}

		go func() {
			f.irc.Kick(channel, nick, reason)
			time.Sleep(100 * time.Millisecond)
			f.irc.Ban(channel, nick)
		}()

		logger.Infof(e, "kickBanned %s in %s", nick, channel)
	})
}
