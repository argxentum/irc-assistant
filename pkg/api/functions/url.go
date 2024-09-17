package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
	"github.com/anaskhan96/soup"
	"math/rand"
	"strings"
)

var userAgents = []string{
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0",
}

const urlFunctionName = "url"

type urlFunction struct {
	Stub
}

func NewURLFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, urlFunctionName)
	if err != nil {
		return nil, err
	}

	return &urlFunction{
		Stub: stub,
	}, nil
}

func (f *urlFunction) MayExecute(e *core.Event) bool {
	if !f.isValid(e, 0) {
		return false
	}

	tokens := Tokens(e.Message())
	return strings.HasPrefix(tokens[0], "https://") || strings.HasPrefix(tokens[0], "http://")
}

func (f *urlFunction) Execute(e *core.Event) {
	tokens := Tokens(e.Message())
	url := tokens[0]

	soup.Header("User-Agent", userAgents[rand.Intn(len(userAgents))])
	resp, err := soup.Get(fmt.Sprintf("%s/%s", "https://nug.zip/", url))
	if err != nil {
		f.Reply(e, "Unable to summarize the URL", e.From)
		return
	}

	doc := soup.HTMLParse(resp)
	title := doc.Find("span", "class", "title")
	if title.Error != nil {
		f.Reply(e, "Unable to summarize the URL", e.From, text.Bold(url))
		return
	}

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s", title.Text()))
}
