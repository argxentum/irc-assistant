package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const joinFunctionName = "join"

type joinFunction struct {
	*functionStub
}

func NewJoinFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &joinFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (f *joinFunction) Name() string {
	return joinFunctionName
}

func (f *joinFunction) Description() string {
	return "Invites to join the specified channel(s)."
}

func (f *joinFunction) Triggers() []string {
	return []string{"join"}
}

func (f *joinFunction) Usages() []string {
	return []string{"%s <channel1> [<channel2> ...]"}
}

func (f *joinFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *joinFunction) CanExecute(e *irc.Event) bool {
	if !e.IsPrivateMessage() || !f.isFunctionEventValid(f, e, 1) {
		return false
	}

	tokens := Tokens(e.Message())
	return irc.IsChannel(tokens[1])
}

func (f *joinFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channels := tokens[1:]

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), strings.Join(channels, ", "))

	for _, channel := range channels {
		if !irc.IsChannel(channel) {
			continue
		}

		f.irc.Join(tokens[1])
		logger.Infof(e, "joined %s", channel)
	}
}
