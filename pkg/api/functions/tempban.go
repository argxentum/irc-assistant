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
	*functionStub
}

func NewTempBanFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &tempBanFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (f *tempBanFunction) Name() string {
	return tempBanFunctionName
}

func (f *tempBanFunction) Description() string {
	return "Temporarily bans the specified user from the channel for the specified duration."
}

func (f *tempBanFunction) Triggers() []string {
	return []string{"tempban", "tb"}
}

func (f *tempBanFunction) Usages() []string {
	return []string{"%s <duration> <mask>"}
}

func (f *tempBanFunction) AllowedInPrivateMessages() bool {
	return false
}

func (f *tempBanFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 2)
}

func (f *tempBanFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := e.ReplyTarget()

	duration := tokens[1]
	mask := tokens[2]

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", f.Name(), e.From, e.ReplyTarget(), channel, mask)

	seconds, err := elapse.ParseDuration(duration)
	if err != nil {
		logger.Errorf(e, "error parsing duration, %s", err)
		f.Replyf(e, "invalid duration, see %s for help", style.Bold(fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, registry.Function(tempBanFunctionName).Triggers()[0])))
		return
	}

	f.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
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
