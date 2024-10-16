package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const echoFunctionName = "echo"

type echoFunction struct {
	*functionStub
}

func NewEchoFunction(ctx context.Context, cfg *config.Config, ircSvc irc.IRC) Function {
	return &echoFunction{
		functionStub: newFunctionStub(ctx, cfg, ircSvc, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (f *echoFunction) Name() string {
	return echoFunctionName
}

func (f *echoFunction) Description() string {
	return "Echoes the given message."
}

func (f *echoFunction) Triggers() []string {
	return []string{"echo"}
}

func (f *echoFunction) Usages() []string {
	return []string{"echo <message>"}
}

func (f *echoFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *echoFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *echoFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], " ")
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), message)
	f.SendMessage(e, e.ReplyTarget(), message)
}
