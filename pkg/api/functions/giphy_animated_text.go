package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"math/rand"
	"strings"
)

const giphyAnimatedTextFunctionName = "giphyAnimatedText"

type giphyAnimatedTextFunction struct {
	FunctionStub
}

func NewGiphyAnimatedTextFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, giphyAnimatedTextFunctionName)
	if err != nil {
		return nil, err
	}

	return &giphyAnimatedTextFunction{
		FunctionStub: stub,
	}, nil
}

func (f *giphyAnimatedTextFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *giphyAnimatedTextFunction) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	message := strings.Join(tokens[1:], " ")
	logger.Infof(e, "⚡ [%s/%s] giphyAnimatedText %s", e.From, e.ReplyTarget(), message)

	res, err := f.giphyAnimatedTextRequest(e, message)
	if err != nil {
		logger.Errorf(e, "error getting giphy animated text, %s", err)
		f.Replyf(e, "unable to perform your request")
		return
	}

	randImage := res.Data[rand.Intn(len(res.Data))]
	f.Replyf(e, randImage.URL)
}
