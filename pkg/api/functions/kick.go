package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const kickFunctionName = "kick"

type kickFunction struct {
	*functionStub
}

func NewKickFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &kickFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (f *kickFunction) Name() string {
	return kickFunctionName
}

func (f *kickFunction) Description() string {
	return "Kicks the specified user from the channel."
}

func (f *kickFunction) Triggers() []string {
	return []string{"kick", "k"}
}

func (f *kickFunction) Usages() []string {
	return []string{"%s <nick> [<reason>]"}
}

func (f *kickFunction) AllowedInPrivateMessages() bool {
	return false
}

func (f *kickFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *kickFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	nick := tokens[1]
	channel := e.ReplyTarget()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", f.Name(), e.From, e.ReplyTarget(), channel, nick)

	f.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			f.Replyf(e, "Missing required permissions to kick users in this channel. Did you forget /mode %s +h %s?", channel, f.cfg.IRC.Nick)
			return
		}

		reason := ""
		if len(tokens) > 2 {
			reason = strings.Join(tokens[2:], " ")
		}
		f.irc.Kick(channel, nick, reason)
		logger.Infof(e, "kicked %s in %s", nick, channel)
	})
}
