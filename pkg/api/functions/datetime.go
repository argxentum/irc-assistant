package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
	"github.com/anaskhan96/soup"
	"math/rand"
	"net/url"
	"strings"
)

const dateTimeFunctionName = "datetime"

type dateTimeFunction struct {
	Stub
}

func NewDateTimeFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, dateTimeFunctionName)
	if err != nil {
		return nil, err
	}

	return &dateTimeFunction{
		Stub: stub,
	}, nil
}

func (f *dateTimeFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *dateTimeFunction) Execute(e *core.Event) {
	tokens := Tokens(e.Message())
	location := strings.Join(tokens[1:], " ")

	soup.Header("User-Agent", userAgents[rand.Intn(len(userAgents))])
	query := url.QueryEscape(fmt.Sprintf("current date and time in %s", location))
	resp, err := soup.Get(fmt.Sprintf("https://www.bing.com/search?q=%s", query))
	if err != nil {
		f.Reply(e, "Unable to find the current date and time of %s", e.From, text.Bold(location))
		return
	}

	doc := soup.HTMLParse(resp)
	label := doc.Find("div", "class", "b_focusLabel")
	time := doc.Find("div", "class", "b_focusTextLarge")
	date := doc.Find("div", "class", "b_secondaryFocus")

	if label.Error != nil || time.Error != nil || date.Error != nil {
		f.Reply(e, "Unable to find the current date and time of %s", e.From, text.Bold(location))
		return
	}

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s: %s on %s", label.Text(), text.Bold(time.Text()), text.Bold(date.Text())))
}
