package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const banFunctionName = "ban"

type banFunction struct {
	*functionStub
}

func NewBanFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &banFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (f *banFunction) Name() string {
	return banFunctionName
}

func (f *banFunction) Description() string {
	return "Bans the given user mask from the channel."
}

func (f *banFunction) Triggers() []string {
	return []string{"ban", "b"}
}

func (f *banFunction) Usages() []string {
	return []string{"%s <mask>"}
}

func (f *banFunction) AllowedInPrivateMessages() bool {
	return false
}

func (f *banFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *banFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	mask := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", f.Name(), e.From, e.ReplyTarget(), channel, mask)

	f.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			f.Replyf(e, "Missing required permissions to ban users in this channel. Did you forget /mode %s +h %s?", channel, f.cfg.IRC.Nick)
			return
		}

		f.irc.Ban(channel, mask)
		logger.Infof(e, "banned %s in %s", mask, channel)
	})
}
