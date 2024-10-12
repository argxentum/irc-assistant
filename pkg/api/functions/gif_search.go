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
	FunctionStub
}

func NewGifSearchFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, gifSearchFunctionName)
	if err != nil {
		return nil, err
	}

	return &gifSearchFunction{
		FunctionStub: stub,
	}, nil
}

func (f *gifSearchFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *gifSearchFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], "_") + ".gif"
	log.Logger().Infof(e, "⚡ [%s/%s] gifSearch %s", e.From, e.ReplyTarget(), message)
	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s/gif/%s", f.cfg.Server.ExternalRootURL, message))
}
