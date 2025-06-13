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

const MarketDataMarketsCommandName = "markets_marketdata"

type MarketDataMarketsCommand struct {
	*commandStub
	retriever retriever.DocumentRetriever
}

func NewMarketDataMarketsCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &MarketDataMarketsCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *MarketDataMarketsCommand) Name() string {
	return MarketDataMarketsCommandName
}

func (c *MarketDataMarketsCommand) Description() string {
	return "Displays current stock market data for the given region. Defaults to US."
}

func (c *MarketDataMarketsCommand) Triggers() []string {
	return []string{"markets", "market"}
}

func (c *MarketDataMarketsCommand) Usages() []string {
	return []string{"%s", "%s <region>"}
}

func (c *MarketDataMarketsCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *MarketDataMarketsCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *MarketDataMarketsCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	region := "US"
	if len(tokens) > 1 {
		region = tokens[1]
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), region)

	message := c.retrieveSummary(e, region)
	if len(message) == 0 {
		logger.Warningf(e, "unable to retrieve stock market information for %s", region)
		c.Replyf(e, "Unable to retrieve stock market information for %s", region)
		return
	}

	c.SendMessage(e, e.ReplyTarget(), message)
}

const marketDataIndicesURL = "https://api.marketdata.app/v1/indices/quotes/%s"

type marketDataSummary struct {
	Status        string    `json:"s"`
	Symbol        []string  `json:"symbol"`
	LastPrice     []float64 `json:"last"`
	Change        []float64 `json:"change"`
	ChangePercent []float64 `json:"changepct"`
	Timestamp     []int64   `json:"updated"`
}

func (c *MarketDataMarketsCommand) retrieveSummary(e *irc.Event, region string) string {
	logger := log.Logger()
	region = strings.TrimSpace(strings.ToUpper(region))

	symbols := []string{"DJI", "IXIC", "SPX", "VIX"}
	names := map[string]string{"DJI": "Dow Jones Industrial Average", "IXIC": "NASDAQ", "SPX": "S&P 500", "VIX": "VIX"}
	summaries := make([]*marketDataSummary, 0)

	for _, symbol := range symbols {
		mr, err := c.retrieveSymbolSummary(e, symbol)
		if err != nil {
			logger.Warningf(e, "unable to retrieve stock market information for %s: %v", symbol, err)
			continue
		}
		if mr == nil {
			logger.Debugf(e, "unable to retrieve stock market information for %s", symbol)
			continue
		}
		summaries = append(summaries, mr)
	}

	if len(summaries) == 0 {
		logger.Warningf(e, "unable to retrieve stock market information for %s", region)
		return ""
	}

	message := ""

	for _, s := range summaries {
		symbol := s.Symbol[0]
		name := names[symbol]
		price := s.LastPrice[0]
		change := s.Change[0]
		changePercent := 100.0 * s.ChangePercent[0]

		if len(symbol) == 0 {
			logger.Warningf(e, "skipping invalid stock market information: %s %.02f %.02f", symbol, price, change)
			return ""
		}

		styledChange := fmt.Sprintf("%.02f", change)
		if change < 0 {
			if changePercent <= -0.01 {
				styledChange = style.ColorForeground(fmt.Sprintf("▼ %.02f (%.02f%%)", change, changePercent), style.ColorRed)
			} else {
				styledChange = style.ColorForeground(fmt.Sprintf("▼ %.02f", change), style.ColorRed)
			}
		} else if change > 0 {
			if changePercent >= 0.01 {
				styledChange = style.ColorForeground(fmt.Sprintf("▲ %.02f (%.02f%%)", change, changePercent), style.ColorGreen)
			} else {
				styledChange = style.ColorForeground(fmt.Sprintf("▲ %.02f", change), style.ColorGreen)
			}
		}

		if len(message) > 0 {
			message += " | "
		}

		message += fmt.Sprintf("%s (%s): %s %s", style.Bold(name), symbol, style.Underline(fmt.Sprintf("%.02f", price)), styledChange)
	}

	return message
}

func (c *MarketDataMarketsCommand) retrieveSymbolSummary(e *irc.Event, symbol string) (*marketDataSummary, error) {
	logger := log.Logger()
	symbol = strings.ToUpper(symbol)

	u := fmt.Sprintf(marketDataIndicesURL, url.QueryEscape(symbol))

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		logger.Warningf(e, "unable to create stock market request for %s: %v", symbol, err)
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.MarketData.APIKey)

	msr, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Warningf(e, "unable to fetch stock overview for symbol %s: %v", symbol, err)
		return nil, err
	}

	defer msr.Body.Close()

	body, err := io.ReadAll(msr.Body)
	if err != nil {
		logger.Warningf(e, "unable to read stock overview for symbol %s: %v", symbol, err)
		return nil, err
	}

	var marketData marketDataSummary
	if err = json.Unmarshal(body, &marketData); err != nil {
		logger.Warningf(e, "unable to parse stock overview for symbol %s: %v", symbol, err)
		return nil, err
	}

	if marketData.Status != "ok" {
		logger.Warningf(e, "status not ok: %v", marketData.Status)
		return nil, errors.New("status not ok: " + marketData.Status)
	}

	return &marketData, nil
}
