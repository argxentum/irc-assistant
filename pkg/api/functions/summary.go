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

const summaryFunctionName = "summary"

type summaryFunction struct {
	Stub
}

func NewSummaryFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, summaryFunctionName)
	if err != nil {
		return nil, err
	}

	return &summaryFunction{
		Stub: stub,
	}, nil
}

func (f *summaryFunction) MayExecute(e *core.Event) bool {
	if !f.isValid(e, 0) {
		return false
	}

	tokens := Tokens(e.Message())
	return strings.HasPrefix(tokens[0], "https://") || strings.HasPrefix(tokens[0], "http://")
}

func (f *summaryFunction) Execute(e *core.Event) {
	fmt.Printf("Executing function: url\n")
	tokens := Tokens(e.Message())
	url := tokens[0]

	soup.Header("User-Agent", userAgents[rand.Intn(len(userAgents))])
	resp, err := soup.Get(fmt.Sprintf("https://nuggetize.com/link/%s", url))
	if err != nil {
		f.Reply(e, "Unable to provide a summary")
		return
	}

	doc := soup.HTMLParse(resp)
	println(doc.HTML())
	title := doc.Find("span", "class", "title")
	if title.Error != nil {
		f.Reply(e, "Unable to provide a summary")
		return
	}

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s", title.Text()))
}
