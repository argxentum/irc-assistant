package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"time"
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
	return f.isValid(e, 2)
}

func (f *tempBanFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()

	duration := tokens[1]
	mask := tokens[2]

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] tempban %s %s", e.From, e.ReplyTarget(), channel, mask)

	seconds, err := elapse.ParseDuration(duration)
	if err != nil {
		logger.Errorf(e, "error parsing duration, %s", err)
		f.Replyf(e, "invalid duration, see %s for help", style.Bold(fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, f.cfg.Functions.EnabledFunctions[tempBanFunctionName].Triggers[0])))
		return
	}

	f.isBotAuthorizedByChannelStatus(channel, irc.HalfOperator, func(authorized bool) {
		if !authorized {
			f.Replyf(e, "Missing required permissions to temporarily ban users in this channel. Did you forget /mode %s +h %s?", channel, f.cfg.IRC.Nick)
			return
		}

		f.irc.Ban(channel, mask)

		task := models.NewBanRemovalTask(time.Now().Add(seconds), mask, channel)
		err = firestore.Get().AddTask(task)
		if err != nil {
			logger.Errorf(e, "error adding task, %s", err)
			return
		}

		logger.Infof(e, "temporarily banned %s from %s for %s", mask, channel, duration)
	})
}
