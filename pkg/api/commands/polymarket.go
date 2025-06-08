package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const PolymarketCommandName = "polymarket"

const PolymarketGammaAPIBaseURL = "https://gamma-api.polymarket.com/markets?active=true&closed=false&order=endDate&ascending=false&limit=%d&offset=%d"
const PolymarketEventPublicURL = "https://polymarket.com/event/%s"
const MaxSearchResults = 10000
const SearchResultLimit = 500

type PolymarketCommand struct {
	*commandStub
}

func NewPolymarketCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &PolymarketCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *PolymarketCommand) Name() string {
	return PolymarketCommandName
}

func (c *PolymarketCommand) Description() string {
	return "Displays the latest Polymarket betting data for the market matching the query."
}

func (c *PolymarketCommand) Triggers() []string {
	return []string{"polymarket", "poly"}
}

func (c *PolymarketCommand) Usages() []string {
	return []string{"%s <query>"}
}

func (c *PolymarketCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *PolymarketCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

type polymarketResult struct {
	ID               string    `json:"id"`
	Question         string    `json:"question"`
	Description      string    `json:"description"`
	Slug             string    `json:"slug"`
	EndDate          string    `json:"endDate"`
	OutcomesRaw      string    `json:"outcomes"`
	Outcomes         []string  `json:"-"`
	OutcomePricesRaw string    `json:"outcomePrices"`
	OutcomePrices    []float64 `json:"-"`
	Volume           float64   `json:"volumeNum"`
	Events           []struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
}

func (c *PolymarketCommand) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	queryTerms := tokens[1:]
	query := strings.Join(queryTerms, " ")
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), query)

	offset := 0
	var result *polymarketResult
	for result == nil && offset < MaxSearchResults {
		var err error
		result, err = findPolymarketResult(offset, queryTerms)
		if err != nil {
			logger.Errorf(e, "error fetching Polymarket results: %s", err)
			c.Replyf(e, "Error fetching Polymarket data")
			return
		}
		if result == nil {
			offset += SearchResultLimit
		}
	}

	if result == nil {
		logger.Warningf(e, "no Polymarket results found for query: %s", query)
		c.Replyf(e, "No active Polymarket results found for %s", style.Bold(query))
		return
	}

	logger.Infof(e, "found Polymarket result: %s", result.ID)

	maxPrice := 0.0
	for _, price := range result.OutcomePrices {
		if price > maxPrice {
			maxPrice = price
		}
	}

	message := style.Bold(result.Question)

	if len(result.Outcomes) == 1 {
		outcome := result.Outcomes[0]
		price := result.OutcomePrices[0]
		trade := fmt.Sprintf("%s $%s", style.Underline(outcome), text.DecorateFloatWithCommas(price))
		if price > 0.5 {
			trade = style.ColorForeground(trade, style.ColorGreen)
		}
		message += " • " + trade
	} else {
		for i, outcome := range result.Outcomes {
			if i == 0 {
				message += " •"
			} else {
				message += " |"
			}
			price := result.OutcomePrices[i]
			trade := fmt.Sprintf(" %s $%s", style.Underline(outcome), text.DecorateFloatWithCommas(price))
			if price == maxPrice {
				message += style.ColorForeground(trade, style.ColorGreen)
			} else {
				message += trade
			}
		}
	}

	if result.Volume > 0 {
		message += fmt.Sprintf(" • Volume: $%s", text.DecorateFloatWithCommas(result.Volume))
	}

	if len(result.EndDate) > 0 {
		endDate, _ := time.Parse(time.RFC3339, result.EndDate)
		if !endDate.IsZero() {
			if len(message) > 0 {
				message += " • "
			}
			message += fmt.Sprintf("Ends %s", elapse.FutureTimeDescription(endDate))
		}
	}

	messages := make([]string, 0)
	messages = append(messages, message)

	if len(result.Events) > 0 && len(result.Events[0].Slug) > 0 {
		messages = append(messages, fmt.Sprintf(PolymarketEventPublicURL, result.Events[0].Slug))
	}

	c.irc.SendMessages(e.ReplyTarget(), messages)
}

func findPolymarketResult(offset int, queryTerms []string) (*polymarketResult, error) {
	url := fmt.Sprintf(PolymarketGammaAPIBaseURL, SearchResultLimit, offset)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching Polymarket results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var results []polymarketResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("error decoding Polymarket results: %w", err)
	}

	for i := range results {
		json.Unmarshal([]byte(results[i].OutcomesRaw), &results[i].Outcomes)
		results[i].OutcomePrices = parseOutcomePrices(results[i].OutcomePricesRaw)
	}

	var match *polymarketResult
	for _, result := range results {
		if len(result.Outcomes) == 0 || len(result.OutcomePrices) == 0 {
			continue
		}

		matches := 0
		for _, term := range queryTerms {
			if strings.Contains(strings.ToLower(result.Question), strings.ToLower(term)) {
				matches++
			}
		}
		if matches == len(queryTerms) {
			match = &result
		}
	}

	return match, nil
}

func parseOutcomePrices(raw string) []float64 {
	if len(raw) == 0 {
		return nil
	}

	var rawPrices []string
	if err := json.Unmarshal([]byte(raw), &rawPrices); err != nil {
		log.Logger().Errorf(nil, "error parsing outcome prices (%s): %s", raw, err)
		return nil
	}

	prices := make([]float64, len(rawPrices))
	for i, price := range rawPrices {
		var err error
		prices[i], err = strconv.ParseFloat(price, 64)
		if err != nil {
			log.Logger().Errorf(nil, "error converting price %s to float64: %s", price, err)
			prices[i] = 0.0 // default to 0.0 if conversion fails
		}
	}

	return prices
}
