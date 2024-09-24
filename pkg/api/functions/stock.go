package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/style"
	"fmt"
	"net/url"
	"strings"
)

const stockFunctionName = "stock"

type stockFunction struct {
	FunctionStub
}

func NewStockFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, stockFunctionName)
	if err != nil {
		return nil, err
	}

	return &stockFunction{
		FunctionStub: stub,
	}, nil
}

func (f *stockFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *stockFunction) Execute(e *core.Event) {
	tokens := Tokens(e.Message())
	symbol := tokens[1]
	fmt.Printf("⚡ stock %s\n", symbol)

	query := url.QueryEscape(fmt.Sprintf("current stock price %s", strings.ToUpper(symbol)))
	doc, err := getDocument(fmt.Sprintf(bingSearchURL, query), true)
	if err != nil {
		f.Reply(e, "Unable to retrieve stock price data for %s.", style.Bold(symbol))
		return
	}

	section := doc.Find("div.finquote").First()
	title := strings.TrimSpace(section.Find("h2").First().Text())
	subtitle := strings.TrimSpace(section.Find("div.fin_metadata").First().Text())
	priceSection := section.Find("div.fin_quotePrice").First()
	price := strings.TrimSpace(priceSection.Find("div#Finance_Quote").First().Text())
	currency := strings.TrimSpace(priceSection.Find("span.price_curr").First().Text())
	change := strings.TrimSpace(priceSection.Find("span.fin_change").First().Text())

	if len(title) == 0 || len(price) == 0 {
		f.Reply(e, "Unable to retrieve stock market information.")
		return
	}

	if len(subtitle) > 0 {
		f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s: %s", style.Bold(title), subtitle))
	} else {
		f.irc.SendMessage(e.ReplyTarget(), style.Bold(title))
	}

	styledChange := change
	if strings.HasPrefix(change, "▼") {
		styledChange = style.ColorForeground(change, style.ColorRed)
	} else if strings.HasPrefix(change, "▲") {
		styledChange = style.ColorForeground(change, style.ColorGreen)
	}

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s %s %s", price, currency, styledChange))
}
