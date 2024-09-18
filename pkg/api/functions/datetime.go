package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
	"github.com/gocolly/colly/v2"
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
	fmt.Printf("⚡ datetime\n")
	tokens := Tokens(e.Message())
	location := strings.Join(tokens[1:], " ")

	c := colly.NewCollector()
	c.UserAgent = userAgents[rand.Intn(len(userAgents))]
	c.OnHTML("div.baselClock", func(node *colly.HTMLElement) {
		label := node.ChildText("div.b_focusLabel")
		time := node.ChildText("div.b_focusTextLarge")
		date := node.ChildText("div.b_secondaryFocus")
		f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s: %s on %s", label, text.Bold(time), text.Bold(date)))
	})

	query := url.QueryEscape(fmt.Sprintf("current date and time in %s", location))
	err := c.Visit(fmt.Sprintf("https://www.bing.com/search?q=%s", query))
	if err != nil {
		f.Reply(e, "Unable to find the current date and time of %s", e.From, text.Bold(location))
		return
	}
}
