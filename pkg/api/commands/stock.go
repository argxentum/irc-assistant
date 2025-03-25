package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

	message, err := c.retrieveStockQuote(e, symbol)
	if len(message) == 0 || err != nil {
		logger.Warningf(e, "unable to retrieve stock price for %s", symbol)
		c.Replyf(e, "Unable to retrieve stock price for %s", symbol)
		return
	}

	c.SendMessage(e, e.ReplyTarget(), message)
}

const alphavantageCompanyOverviewURL = "https://www.alphavantage.co/query?function=OVERVIEW&symbol=%s&apikey=%s"
const finnhubCompanyOverviewURL = "https://finnhub.io/api/v1/stock/profile2?symbol=%s&token=%s"
const finnhubStockMetricsURL = "https://finnhub.io/api/v1/stock/metric?symbol=%s&metric=all&token=%s"
const finnhubStockQuoteURL = "https://finnhub.io/api/v1/quote?symbol=%s&token=%s"

type companyOverview struct {
	symbol     string
	name       string
	currency   string
	country    string
	exchange   string
	high52Week string
	low52Week  string
}
type alphavantageCompanyOverview struct {
	Symbol     string `json:"Symbol"`
	Name       string `json:"Name"`
	Currency   string `json:"Currency"`
	Country    string `json:"Country"`
	Exchange   string `json:"Exchange"`
	High52Week string `json:"52WeekHigh"`
	Low52Week  string `json:"52WeekLow"`
}

type finnhubCompanyOverview struct {
	Symbol   string `json:"ticker"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Country  string `json:"country"`
	Exchange string `json:"exchange"`
}

type finnhubStockMetrics struct {
	Metric struct {
		High52Week float64 `json:"52WeekHigh"`
		Low52Week  float64 `json:"52WeekLow"`
	} `json:"metric"`
}

type finnhubStockQuote struct {
	CurrentPrice  float64 `json:"c"`
	Change        float64 `json:"d"`
	PercentChange float64 `json:"dp"`
	DayHigh       float64 `json:"h"`
	DayLow        float64 `json:"l"`
	DayOpen       float64 `json:"o"`
	PreviousClose float64 `json:"pc"`
	Timestamp     int64   `json:"t"`
}

func (c *StockCommand) retrieveStockQuote(e *irc.Event, symbol string) (string, error) {
	logger := log.Logger()
	symbol = strings.ToUpper(symbol)

	overview, err := c.retrieveCompanyOverview(e, symbol)
	if err != nil {
		return "", err
	}

	u := fmt.Sprintf(finnhubStockQuoteURL, url.QueryEscape(symbol), c.cfg.Finnhub.APIKey)
	logger.Debugf(e, "making stock quote request: %s", u)

	fhr, err := http.Get(u)
	if err != nil {
		logger.Warningf(e, "error fetching finnhub quote for symbol %s: %v", symbol, err)
		return "", err
	}

	defer fhr.Body.Close()

	body, err := io.ReadAll(fhr.Body)
	if err != nil {
		logger.Warningf(e, "error reading finnhub response for %s: %v", symbol, err)
		return "", err
	}

	var quote finnhubStockQuote
	if err = json.Unmarshal(body, &quote); err != nil {
		logger.Warningf(e, "error parsing finnhub response for %s: %v", symbol, err)
		return "", err
	}

	if len(overview.name) == 0 || quote.CurrentPrice == 0 {
		logger.Warningf(e, "unable to parse stock market information, title: %s, price: %.02f", overview.name, quote.CurrentPrice)
		return "", errors.New("unable to parse stock market information")
	}

	subtitle := fmt.Sprintf("%s: %s", overview.exchange, overview.symbol)

	message := fmt.Sprintf("%s – ", style.Bold(overview.name))
	if len(subtitle) > 0 {
		message = fmt.Sprintf("%s (%s) – ", style.Bold(overview.name), subtitle)
	}

	styledChange := fmt.Sprintf("%.02f", quote.Change)
	if quote.Change < 0 {
		styledChange = style.ColorForeground(fmt.Sprintf("▼ %.02f (%.02f%%)", quote.Change, quote.PercentChange), style.ColorRed)
	} else if quote.Change > 0 {
		styledChange = style.ColorForeground(fmt.Sprintf("▲ %.02f (%.02f%%)", quote.Change, quote.PercentChange), style.ColorGreen)
	}

	message += fmt.Sprintf("%s | %s", style.Underline(fmt.Sprintf("%.02f %s", quote.CurrentPrice, overview.currency)), styledChange)
	message += fmt.Sprintf(" | Open: %.02f", quote.DayOpen)
	message += fmt.Sprintf(" | High: %.02f", quote.DayHigh)
	message += fmt.Sprintf(" | Low: %.02f", quote.DayLow)
	message += fmt.Sprintf(" | 52W High: %s", overview.high52Week)
	message += fmt.Sprintf(" | 52W Low: %s", overview.low52Week)

	return message, nil
}

func (c *StockCommand) retrieveCompanyOverview(e *irc.Event, symbol string) (*companyOverview, error) {
	logger := log.Logger()

	u := fmt.Sprintf(alphavantageCompanyOverviewURL, url.QueryEscape(symbol), c.cfg.Alphavantage.APIKey)
	logger.Debugf(e, "making stock overview request: %s", u)

	avr, err := http.Get(u)
	if err != nil {
		logger.Warningf(e, "unable to fetch stock overview for symbol %s: %v", symbol, err)
		return nil, err
	}

	defer avr.Body.Close()

	body, err := io.ReadAll(avr.Body)
	if err != nil {
		logger.Warningf(e, "unable to read stock overview for symbol %s: %v", symbol, err)
		return nil, err
	}

	var overview *companyOverview

	var avOverview alphavantageCompanyOverview
	if err = json.Unmarshal(body, &avOverview); err != nil {
		logger.Debugf(e, "unable to parse stock overview for symbol %s: %v", symbol, err)
	}

	if len(avOverview.Name) == 0 {
		u = fmt.Sprintf(finnhubCompanyOverviewURL, url.QueryEscape(symbol), c.cfg.Finnhub.APIKey)
		logger.Debugf(e, "making company overview request: %s", u)

		fhor, err := http.Get(u)
		if err != nil {
			logger.Warningf(e, "unable to fetch company overview for symbol %s: %v", symbol, err)
			return nil, err
		}

		defer fhor.Body.Close()

		body, err = io.ReadAll(fhor.Body)
		if err != nil {
			logger.Warningf(e, "unable to read company overview for symbol %s: %v", symbol, err)
			return nil, err
		}

		var fhOverview finnhubCompanyOverview
		if err = json.Unmarshal(body, &fhOverview); err != nil {
			logger.Warningf(e, "unable to parse company overview for symbol %s: %v", symbol, err)
			return nil, err
		}

		overview = &companyOverview{
			symbol:   fhOverview.Symbol,
			name:     fhOverview.Name,
			currency: fhOverview.Currency,
			country:  fhOverview.Country,
		}

		if strings.Contains(strings.ToLower(fhOverview.Exchange), "nyse") || strings.Contains(strings.ToLower(fhOverview.Exchange), "new york stock exchange") {
			overview.exchange = "NYSE"
		} else if strings.Contains(strings.ToLower(fhOverview.Exchange), "nasdaq") {
			overview.exchange = "NASDAQ"
		} else {
			overview.exchange = "OTHER"
		}

		u = fmt.Sprintf(finnhubStockMetricsURL, url.QueryEscape(symbol), c.cfg.Finnhub.APIKey)
		logger.Debugf(e, "making company overview request: %s", u)

		fhsr, err := http.Get(u)
		if err != nil {
			logger.Warningf(e, "unable to fetch company overview for symbol %s: %v", symbol, err)
			return nil, err
		}

		defer fhsr.Body.Close()

		body, err = io.ReadAll(fhsr.Body)
		if err != nil {
			logger.Warningf(e, "unable to read company overview for symbol %s: %v", symbol, err)
			return nil, err
		}

		var fhMetrics finnhubStockMetrics
		if err = json.Unmarshal(body, &fhMetrics); err != nil {
			logger.Warningf(e, "unable to parse company overview for symbol %s: %v", symbol, err)
			return nil, err
		}

		overview.high52Week = fmt.Sprintf("%.02f", fhMetrics.Metric.High52Week)
		overview.low52Week = fmt.Sprintf("%.02f", fhMetrics.Metric.Low52Week)
	} else {
		overview = &companyOverview{
			symbol:     avOverview.Symbol,
			name:       avOverview.Name,
			currency:   avOverview.Currency,
			country:    avOverview.Country,
			exchange:   avOverview.Exchange,
			high52Week: avOverview.High52Week,
			low52Week:  avOverview.Low52Week,
		}
	}

	return overview, nil
}

func (c *StockCommand) retrieveBingQuoteCardStockPrice(e *irc.Event, symbol string) (string, error) {
	logger := log.Logger()

	query := url.QueryEscape(fmt.Sprintf("current stock price %s", strings.ToUpper(symbol)))
	params := retriever.DefaultParams(fmt.Sprintf(bingSearchURL, query))

	section, err := c.retriever.RetrieveDocumentSelection(e, params, "html")
	if err != nil {
		logger.Warningf(e, "unable to retrieve stock price data for %s: %s", symbol, err)
		return "", err
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
		return "", errors.New("unable to parse stock market information")
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

	return message, nil
}
