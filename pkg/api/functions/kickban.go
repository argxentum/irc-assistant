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
	*functionStub
}

func NewKickBanFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &kickBanFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (f *kickBanFunction) Name() string {
	return kickBanFunctionName
}

func (f *kickBanFunction) Description() string {
	return "Kicks and bans the specified user from the channel."
}

func (f *kickBanFunction) Triggers() []string {
	return []string{"kickban", "kb"}
}

func (f *kickBanFunction) Usages() []string {
	return []string{"%s <nick> [<reason>]"}
}

func (f *kickBanFunction) AllowedInPrivateMessages() bool {
	return false
}

func (f *kickBanFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
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
	logger.Infof(e, "⚡ %s [%s/%s] %s %s - %s", f.Name(), e.From, e.ReplyTarget(), channel, nick, reason)

	f.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
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
