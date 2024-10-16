package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"net/url"
	"strings"
)

const stockFunctionName = "stock"

type stockFunction struct {
	*functionStub
	retriever retriever.DocumentRetriever
}

func NewStockFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) Function {
	return &stockFunction{
		functionStub: defaultFunctionStub(ctx, cfg, irc),
		retriever:    retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (f *stockFunction) Name() string {
	return stockFunctionName
}

func (f *stockFunction) Description() string {
	return "Displays the current price of the given stock symbol or company name."
}

func (f *stockFunction) Triggers() []string {
	return []string{"stock"}
}

func (f *stockFunction) Usages() []string {
	return []string{"%s <symbol>"}
}

func (f *stockFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *stockFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *stockFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	symbol := tokens[1]

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), symbol)

	message := f.retrieveStockPriceMessage(e, symbol)
	if len(message) == 0 {
		logger.Warningf(e, "unable to retrieve stock price for %s", symbol)
		f.Replyf(e, "Unable to retrieve stock price for %s", symbol)
		return
	}

	f.SendMessage(e, e.ReplyTarget(), message)
}

func (f *stockFunction) retrieveStockPriceMessage(e *irc.Event, symbol string) string {
	logger := log.Logger()

	query := url.QueryEscape(fmt.Sprintf("current stock price %s", strings.ToUpper(symbol)))
	section, err := f.retriever.RetrieveDocumentSelection(e, retriever.DefaultParams(fmt.Sprintf(bingSearchURL, query)), "div.finquote")
	if err != nil {
		logger.Warningf(e, "unable to retrieve stock price data for %s: %s", symbol, err)
		return ""
	}

	title := strings.TrimSpace(section.Find("h2").First().Text())
	subtitle := strings.TrimSpace(section.Find("div.fin_metadata").First().Text())
	priceSection := section.Find("div.fin_quotePrice").First()
	price := strings.TrimSpace(priceSection.Find("div#Finance_Quote").First().Text())
	currency := strings.TrimSpace(priceSection.Find("span.price_curr").First().Text())
	change := strings.TrimSpace(priceSection.Find("span.fin_change").First().Text())

	if len(title) == 0 || len(price) == 0 {
		logger.Warningf(e, "unable to parse stock market information, title: %s, price: %s", title, price)
		return ""
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
	return message
}
