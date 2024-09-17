package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
	"github.com/anaskhan96/soup"
	"math/rand"
	"strings"
)

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
	fmt.Printf("Executing function: url\n")
	tokens := Tokens(e.Message())
	url := tokens[0]

	soup.Header("User-Agent", userAgents[rand.Intn(len(userAgents))])
	resp, err := soup.Get(fmt.Sprintf("%s/%s", "https://nug.zip/", url))
	if err != nil {
		f.Reply(e, "Unable to summarize the URL")
		return
	}

	doc := soup.HTMLParse(resp)
	title := doc.Find("span", "class", "title")
	if title.Error != nil {
		f.Reply(e, "Unable to summarize the URL")
		return
	}

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s", title.Text()))
}
