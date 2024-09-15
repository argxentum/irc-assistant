package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
	"github.com/anaskhan96/soup"
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

func (f *dateTimeFunction) Matches(e *core.Event) bool {
	if !f.isAuthorized(e) {
		return false
	}

	tokens := sanitizedTokens(e.Message(), 200)
	if len(tokens) < 2 {
		return false
	}

	for _, p := range f.Prefixes {
		if tokens[0] == p {
			return true
		}
	}
	return false
}

func (f *dateTimeFunction) Execute(e *core.Event) error {
	tokens := sanitizedTokens(e.Message(), 200)
	location := strings.Join(tokens[1:], " ")

	soup.Header("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	query := strings.Replace(fmt.Sprintf("current date and time in %s", location), " ", "%20", -1)
	resp, err := soup.Get(fmt.Sprintf("https://www.bing.com/search?q=%s", query))
	if err != nil {
		return err
	}

	label := soup.HTMLParse(resp).Find("div", "class", "b_focusLabel").Text()
	time := soup.HTMLParse(resp).Find("div", "class", "b_focusTextLarge").Text()
	date := soup.HTMLParse(resp).Find("div", "class", "b_secondaryFocus").Text()

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s: %s on %s", label, text.Bold(time), text.Bold(date)))
	return nil
}
