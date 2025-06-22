package commands

import (
	"assistant/pkg/api/irc"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const PolymarketGammaAPIEventsURL = "https://gamma-api.polymarket.com/events?slug=%s&limit=1"

func (c *SummaryCommand) parsePolymarket(e *irc.Event, url string) (*summary, error) {
	uc := strings.Split(url, "/")
	if i := strings.Index(uc[len(uc)-1], "?"); i != -1 {
		uc[len(uc)-1] = uc[len(uc)-1][:i]
	}
	slug := uc[len(uc)-1]

	if len(slug) == 0 {
		return nil, nil
	}

	result, err := findPolymarketSlugMarketResult(slug)
	if err != nil {
		return nil, fmt.Errorf("error finding Polymarket market result: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	messages := generatePolymarketMessages(result)

	return &summary{
		messages: messages,
	}, nil
}

func findPolymarketSlugMarketResult(slug string) (*polymarketMarketResult, error) {
	url := fmt.Sprintf(PolymarketGammaAPIEventsURL, slug)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching Polymarket results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var results []polymarketEventResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("error decoding Polymarket results: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	result := results[0]

	if len(result.Markets) == 0 {
		return nil, nil
	}

	market := result.Markets[0]
	json.Unmarshal([]byte(market.OutcomesRaw), &market.Outcomes)
	market.OutcomePrices = parseOutcomePrices(market.OutcomePricesRaw)

	return &market, nil
}
