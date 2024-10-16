package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const sayFunctionName = "say"

type sayFunction struct {
	*functionStub
}

func NewSayFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &sayFunction{
		functionStub: newFunctionStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (f *sayFunction) Name() string {
	return sayFunctionName
}

func (f *sayFunction) Description() string {
	return "Sends a message to the specified channel."
}

func (f *sayFunction) Triggers() []string {
	return []string{"say"}
}

func (f *sayFunction) Usages() []string {
	return []string{"%s <channel> <message>"}
}

func (f *sayFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *sayFunction) CanExecute(e *irc.Event) bool {
	if !f.isFunctionEventValid(f, e, 3) {
		return false
	}

	tokens := Tokens(e.Message())
	return irc.IsChannel(tokens[1])
}

func (f *sayFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := tokens[1]
	message := strings.Join(tokens[2:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", f.Name(), e.From, e.ReplyTarget(), channel, message)

	f.SendMessage(e, channel, message)
}
