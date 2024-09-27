package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"net/url"
	"strings"
)

const stockFunctionName = "stock"

type stockFunction struct {
	FunctionStub
}

func NewStockFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, stockFunctionName)
	if err != nil {
		return nil, err
	}

	return &stockFunction{
		FunctionStub: stub,
	}, nil
}

func (f *stockFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *stockFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	symbol := tokens[1]

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] stock %s", e.From, e.ReplyTarget(), symbol)

	query := url.QueryEscape(fmt.Sprintf("current stock price %s", strings.ToUpper(symbol)))
	doc, err := getDocument(fmt.Sprintf(bingSearchURL, query), true)
	if err != nil {
		logger.Warningf(e, "unable to retrieve stock price data for %s: %s", symbol, err)
		f.Replyf(e, "Unable to retrieve stock price data for %s.", style.Bold(symbol))
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
		logger.Warningf(e, "unable to parse stock market information, title: %s, price: %s", title, price)
		f.Replyf(e, "Unable to retrieve stock market information.")
		return
	}

	message := fmt.Sprintf("%s – ", style.Bold(title))
	if len(subtitle) > 0 {
		message = fmt.Sprintf("%s (%s) – ", style.Bold(title), subtitle)
	}

	styledChange := change
	if strings.HasPrefix(change, "▼") {
		styledChange = style.ColorForeground(change, style.ColorRed)
	} else if strings.HasPrefix(change, "▲") {
		styledChange = style.ColorForeground(change, style.ColorGreen)
	}

	message += fmt.Sprintf("%s %s %s", price, currency, styledChange)
	f.SendMessage(e, e.ReplyTarget(), message)
}
