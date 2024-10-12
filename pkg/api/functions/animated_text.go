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
	FunctionStub
}

func NewAnimatedTextFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, animatedTextFunctionName)
	if err != nil {
		return nil, err
	}

	return &animatedTextFunction{
		FunctionStub: stub,
	}, nil
}

func (f *animatedTextFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *animatedTextFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], "_") + ".gif"
	log.Logger().Infof(e, "⚡ [%s/%s] text %s", e.From, e.ReplyTarget(), message)
	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s/text/%s", f.cfg.Web.ExternalRootURL, message))
}
