package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const gifSearchFunctionName = "gifSearch"

type gifSearchFunction struct {
	*functionStub
}

func NewGifSearchFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) Function {
	return &gifSearchFunction{
		functionStub: defaultFunctionStub(ctx, cfg, irc),
	}
}

func (f *gifSearchFunction) Name() string {
	return gifSearchFunctionName
}

func (f *gifSearchFunction) Description() string {
	return "Searches for the specified gif."
}

func (f *gifSearchFunction) Triggers() []string {
	return []string{"gif"}
}

func (f *gifSearchFunction) Usages() []string {
	return []string{"%s <search>"}
}

func (f *gifSearchFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *gifSearchFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *gifSearchFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], "_") + ".gif"
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), message)
	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s/gifs/%s", f.cfg.Web.ExternalRootURL, message))
}
