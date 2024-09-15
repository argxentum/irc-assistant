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

const dateTimeFunctionName = "datetime"

type dateTimeFunction struct {
	stub
}

func NewDateTimeFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, dateTimeFunctionName)
	if err != nil {
		return nil, err
	}

	return &dateTimeFunction{
		stub: stub,
	}, nil
}

func (f *dateTimeFunction) ShouldExecute(e *core.Event) bool {
	ok, _ := f.verifyInput(e, 1)
	return ok
}

var userAgents = []string{
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0",
}

func (f *dateTimeFunction) Execute(e *core.Event) error {
	tokens := parseTokens(e.Message())
	location := strings.Join(tokens[1:], " ")

	soup.Header("User-Agent", userAgents[rand.Intn(len(userAgents))])
	query := strings.Replace(fmt.Sprintf("current date and time in %s", location), " ", "%20", -1)
	resp, err := soup.Get(fmt.Sprintf("https://www.bing.com/search?q=%s", query))
	if err != nil {
		return err
	}

	label := soup.HTMLParse(resp).Find("div", "class", "b_focusLabel")
	time := soup.HTMLParse(resp).Find("div", "class", "b_focusTextLarge")
	date := soup.HTMLParse(resp).Find("div", "class", "b_secondaryFocus")

	if label.Error != nil || time.Error != nil || date.Error != nil {
		f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("Could not find the current date and time of %s", text.Bold(location)))
		return nil
	}

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s: %s on %s", label.Text(), text.Bold(time.Text()), text.Bold(date.Text())))
	return nil
}
