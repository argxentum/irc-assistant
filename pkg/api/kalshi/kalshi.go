package kalshi

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const apiEventURL = "http://api.elections.kalshi.com/trade-api/v2/events/%s"
const apiEventsURL = "https://api.elections.kalshi.com/trade-api/v2/events?status=open&limit=%d&cursor="
const apiMarketsURL = "https://api.elections.kalshi.com/trade-api/v2/markets?event_ticker=%s&limit=%d"
const maxSearchResults = 10000
const searchResultLimit = 200
const maxMarketsDisplayed = 4
const eventPublicURL = "https://kalshi.com/markets/%s/%s"

type Event struct {
	EventTicker  string `json:"event_ticker"`
	SeriesTicker string `json:"series_ticker"`
	Title        string `json:"title"`
	Subtitle     string `json:"sub_title"`
}

type Market struct {
	Ticker             string  `json:"ticker"`
	EventTicker        string  `json:"event_ticker"`
	MarketType         string  `json:"market_type"`
	Title              string  `json:"title"`
	Subtitle           string  `json:"subtitle"`
	YesSubtitle        string  `json:"yes_sub_title"`
	NoSubtitle         string  `json:"no_sub_title"`
	Expiration         string  `json:"expiration_time"`
	ResponsePriceUnits string  `json:"response_price_units"`
	YesPrice           float64 `json:"yes_ask"`
	NoPrice            float64 `json:"no_ask"`
	Volume             float64 `json:"volume"`
	Liquidity          float64 `json:"liquidity"`
	RulesPrimary       string  `json:"rules_primary"`
	RulesSecondary     string  `json:"rules_secondary"`
}

func GetEventAndMarkets(eventTicker string) (*Event, []*Market, error) {
	logger := log.Logger()

	eventTicker = strings.ToUpper(eventTicker)
	u := fmt.Sprintf(apiEventURL, eventTicker)
	logger.Debugf(nil, "Kalshi event request: %s", u)

	resp, err := http.Get(u)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching Kalshi event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("error fetching Kalshi event: %s", resp.Status)
	}

	var response struct {
		Event   *Event    `json:"event"`
		Markets []*Market `json:"markets"`
		Cursor  string    `json:"cursor"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, nil, fmt.Errorf("error parsing Kalshi events: %w", err)
	}

	return response.Event, response.Markets, nil
}

func FindEvent(query, cursor string, results int) (*Event, error) {
	logger := log.Logger()

	if results >= maxSearchResults {
		return nil, nil
	}

	u := fmt.Sprintf(apiEventsURL, searchResultLimit) + cursor
	logger.Debugf(nil, "Kalshi events request: %s", u)

	resp, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("error fetching Kalshi events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching Kalshi events: %s", resp.Status)
	}

	var response struct {
		Events []*Event `json:"events"`
		Cursor string   `json:"cursor"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error parsing Kalshi events: %w", err)
	}

	if len(response.Events) == 0 {
		return nil, nil
	}

	queryKeywords := text.ParseKeywords(query)
	for _, result := range response.Events {
		if text.ContainsAll(strings.ToLower(result.Title), queryKeywords) {
			return result, nil
		}
	}

	if response.Cursor == "" {
		return nil, nil
	}

	return FindEvent(query, response.Cursor, results+len(response.Events))
}

func FindMarkets(eventTicker string) ([]*Market, error) {
	logger := log.Logger()

	eventTicker = strings.ToUpper(eventTicker)
	u := fmt.Sprintf(apiMarketsURL, eventTicker, searchResultLimit)
	logger.Debugf(nil, "Kalshi markets request: %s", u)

	resp, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("error fetching Kalshi market results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching Kalshi markets: %s", resp.Status)
	}

	var response struct {
		Markets []*Market `json:"markets"`
		Cursor  string    `json:"cursor"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding Kalshi results: %w", err)
	}

	sort.Slice(response.Markets, func(i, j int) bool {
		return response.Markets[i].YesPrice > response.Markets[j].YesPrice
	})

	return response.Markets, nil
}

func Summarize(event *Event, markets []*Market, includeEventURL bool) []string {
	message := style.Bold(event.Title)

	if len(markets) == 0 {
		return nil
	}

	if len(markets) == 1 {
		yesPrice := markets[0].YesPrice / 100.0
		yesTrade := fmt.Sprintf("%s $%s", style.Underline("Yes"), text.DecorateFloatWithCommas(yesPrice))

		noPrice := markets[0].NoPrice / 100.0
		noTrade := fmt.Sprintf("%s $%s", style.Underline("No"), text.DecorateFloatWithCommas(noPrice))

		if yesPrice != noPrice {
			if yesPrice > 0.5 {
				yesTrade = style.ColorForeground(yesTrade, style.ColorGreen)
			} else {
				noTrade = style.ColorForeground(noTrade, style.ColorGreen)
			}
		}

		trade := fmt.Sprintf("%s | %s", yesTrade, noTrade)
		message += " • " + trade
	} else {
		maxPrice := 0.0
		for _, market := range markets {
			if market.YesPrice > maxPrice {
				maxPrice = market.YesPrice
			}
		}

		truncated := 0
		if len(markets) > maxMarketsDisplayed {
			truncated = len(markets) - maxMarketsDisplayed
			markets = markets[:maxMarketsDisplayed]
		}

		for i, market := range markets {
			if i == 0 {
				message += " •"
			} else {
				message += " |"
			}

			price := market.YesPrice / 100.0
			trade := fmt.Sprintf(" %s $%s", style.Underline(market.YesSubtitle), text.DecorateFloatWithCommas(price))
			if market.YesPrice == maxPrice {
				message += style.ColorForeground(trade, style.ColorGreen)
			} else {
				message += trade
			}
		}

		if truncated > 0 {
			plural := "s"
			if truncated == 1 {
				plural = ""
			}
			message += fmt.Sprintf(" (+%d other outcome%s)", truncated, plural)
		}
	}

	if markets[0].Volume > 0 {
		message += fmt.Sprintf(" • Volume: $%s", text.DecorateFloatWithCommas(markets[0].Volume))
	}

	if len(markets[0].Expiration) > 0 {
		endDate, _ := time.Parse(time.RFC3339, markets[0].Expiration)
		if !endDate.IsZero() {
			if len(message) > 0 {
				message += " • "
			}
			message += fmt.Sprintf("Ends %s", elapse.FutureTimeDescription(endDate))
		}
	}

	messages := make([]string, 0)
	messages = append(messages, message)

	if includeEventURL && len(event.EventTicker) > 0 && len(event.SeriesTicker) > 0 {
		messages = append(messages, fmt.Sprintf(eventPublicURL, event.SeriesTicker, event.EventTicker))
	}

	return messages
}
