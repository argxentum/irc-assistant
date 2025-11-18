package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const PolymarketGammaAPIEventsURL = "https://gamma-api.polymarket.com/events?slug=%s&limit=1"

func (c *SummaryCommand) parsePolymarket(e *irc.Event, url string) (*summary, *models.Source, error) {
	uc := strings.Split(url, "/")
	if i := strings.Index(uc[len(uc)-1], "?"); i != -1 {
		uc[len(uc)-1] = uc[len(uc)-1][:i]
	}
	slug := uc[len(uc)-1]

	if len(slug) == 0 {
		return nil, nil, nil
	}

	result, total, err := findPolymarketSlugMarketResult(slug)
	if err != nil {
		return nil, nil, fmt.Errorf("error finding Polymarket market result: %w", err)
	}

	if result == nil {
		return nil, nil, nil
	}

	messages := generatePolymarketMessages(result, total)

	return &summary{
		messages: messages,
	}, nil, nil
}

func findPolymarketSlugMarketResult(slug string) (*polymarketMarketResult, int, error) {
	url := fmt.Sprintf(PolymarketGammaAPIEventsURL, slug)
	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, fmt.Errorf("error fetching Polymarket results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var results []polymarketEventResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, 0, fmt.Errorf("error decoding Polymarket results: %w", err)
	}

	if len(results) == 0 {
		return nil, 0, nil
	}

	result := results[0]

	if len(result.Markets) == 0 {
		return nil, 0, nil
	}

	markets := make([]*polymarketMarketResult, 0, len(result.Markets))
	for _, m := range result.Markets {
		if len(m.OutcomePricesRaw) > 0 {
			markets = append(markets, &m)
		}
	}

	if len(markets) == 0 {
		return nil, 0, nil
	}

	var maxMarket *polymarketMarketResult
	maxOutcomePrice := 0.0
	for _, market := range markets {
		json.Unmarshal([]byte(market.OutcomesRaw), &market.Outcomes)
		market.OutcomePrices = parseOutcomePrices(market.OutcomePricesRaw)
		if len(market.OutcomePrices) > 0 && market.OutcomePrices[0] > maxOutcomePrice {
			maxOutcomePrice = market.OutcomePrices[0]
			maxMarket = market
		}
	}

	return maxMarket, len(markets), nil
}
