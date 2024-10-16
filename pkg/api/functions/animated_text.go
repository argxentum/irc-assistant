package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const animatedTextFunctionName = "animatedText"

type animatedTextFunction struct {
	*functionStub
}

func NewAnimatedTextFunction(ctx context.Context, cfg *config.Config, ircs irc.IRC) Function {
	return &animatedTextFunction{
		functionStub: defaultFunctionStub(ctx, cfg, ircs),
	}
}

func (f *animatedTextFunction) Name() string {
	return animatedTextFunctionName
}

func (f *animatedTextFunction) Description() string {
	return "Displays the given text as an animation."
}

func (f *animatedTextFunction) Triggers() []string {
	return []string{"animate", "animated", "text"}
}

func (f *animatedTextFunction) Usages() []string {
	return []string{"%s <text>"}
}

func (f *animatedTextFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *animatedTextFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *animatedTextFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], "_") + ".gif"
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), message)
	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s/animated/%s", f.cfg.Web.ExternalRootURL, message))
}
