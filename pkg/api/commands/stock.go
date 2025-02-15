package commands

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

const StockCommandName = "stock"

type StockCommand struct {
	*commandStub
	retriever retriever.DocumentRetriever
}

func NewStockCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &StockCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *StockCommand) Name() string {
	return StockCommandName
}

func (c *StockCommand) Description() string {
	return "Displays the current price of the given stock symbol or company name."
}

func (c *StockCommand) Triggers() []string {
	return []string{"stock"}
}

func (c *StockCommand) Usages() []string {
	return []string{"%s <symbol>", "%s <company name>"}
}

func (c *StockCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *StockCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *StockCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	symbol := strings.Join(tokens[1:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), symbol)

	message := c.retrieveStockPriceMessage(e, symbol)
	if len(message) == 0 {
		logger.Warningf(e, "unable to retrieve stock price for %s", symbol)
		c.Replyf(e, "Unable to retrieve stock price for %s", symbol)
		return
	}

	c.SendMessage(e, e.ReplyTarget(), message)
}

func (c *StockCommand) retrieveStockPriceMessage(e *irc.Event, symbol string) string {
	logger := log.Logger()

	query := url.QueryEscape(fmt.Sprintf("current stock price %s", strings.ToUpper(symbol)))
	section, err := c.retriever.RetrieveDocumentSelection(e, retriever.DefaultParams(fmt.Sprintf(bingSearchURL, query)), "html")
	if err != nil {
		logger.Warningf(e, "unable to retrieve stock price data for %s: %s", symbol, err)
		return ""
	}

	title := strings.TrimSpace(section.Find("div.enti_ttl").Text())
	if len(title) == 0 {
		title = strings.TrimSpace(section.Find("h2").First().Text())
	}

	subtitle := strings.TrimSpace(section.Find("div.enti_stxt").Text())
	if len(subtitle) == 0 {
		subtitle = strings.TrimSpace(section.Find("div.fin_metadata").First().Text())
	}

	priceSection := section.Find("div.fin_quoteCard").First()
	price := strings.TrimSpace(priceSection.Find("div#Finance_Quote").First().Text())
	currency := strings.TrimSpace(priceSection.Find("span.price_curr").First().Text())
	change := strings.TrimSpace(priceSection.Find("span.fin_change").First().Text())
	open := strings.TrimSpace(priceSection.Find("div[data-partnertag='Finance.Open']").Next().Text())
	high := strings.TrimSpace(priceSection.Find("div[data-partnertag='Finance.High']").Next().Text())
	low := strings.TrimSpace(priceSection.Find("div[data-partnertag='Finance.Low']").Next().Text())
	yearHigh := strings.TrimSpace(priceSection.Find("div[data-partnertag='Finance.YeahHigh']").Next().Text())
	yearLow := strings.TrimSpace(priceSection.Find("div[data-partnertag='Finance.YeahLow']").Next().Text())

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

	if len(open) > 0 {
		message += fmt.Sprintf(" | Open: %s", open)
	}

	if len(high) > 0 {
		message += fmt.Sprintf(" | High: %s", high)
	}

	if len(low) > 0 {
		message += fmt.Sprintf(" | Low: %s", low)
	}

	if len(yearHigh) > 0 {
		message += fmt.Sprintf(" | 52W High: %s", yearHigh)
	}

	if len(yearLow) > 0 {
		message += fmt.Sprintf(" | 52W Low: %s", yearLow)
	}

	return message
}
