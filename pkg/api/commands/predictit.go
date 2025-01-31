package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const PredictItCommandName = "predictIt"

const predictItSearchBaseURL = "https://www.predictit.org/api/Browse/Search"
const predictItMarketURL = "https://www.predictit.org/api/marketdata/markets/%d"
const predictItMarketDetailURL = "https://www.predictit.org/markets/detail/%d"

type PredictItCommand struct {
	*commandStub
}

func NewPredictItCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &PredictItCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *PredictItCommand) Name() string {
	return PredictItCommandName
}

func (c *PredictItCommand) Description() string {
	return "Displays the latest PredictIt betting data for the market matching the query."
}

func (c *PredictItCommand) Triggers() []string {
	return []string{"predictit", "betting"}
}

func (c *PredictItCommand) Usages() []string {
	return []string{"%s <query>"}
}

func (c *PredictItCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *PredictItCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

type predictItSearchResult struct {
	Markets []struct {
		ID        int    `json:"marketId"`
		Name      string `json:"marketName"`
		Status    string `json:"status"`
		Contracts []struct {
			ID     int     `json:"contractId"`
			Name   string  `json:"contractName"`
			Price  float64 `json:"lastTradePrice"`
			Trades int     `json:"totalTrades"`
		} `json:"contracts"`
	}
}

type predictItMarketResult struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	URL       string `json:"url"`
	Contracts []struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Price  float64 `json:"lastTradePrice"`
		Trades int     `json:"totalTrades"`
	} `json:"contracts"`
}

func (c *PredictItCommand) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	input := strings.Join(tokens[1:], " ")
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), input)

	baseURL, err := url.Parse(predictItSearchBaseURL)
	if err != nil {
		logger.Warningf(e, "malformed URL: %s", err)
		return
	}

	illegal := []string{"?", "<", ">", "|", "\\", "/", ":", "*", "\""}
	for _, i := range illegal {
		input = strings.ReplaceAll(input, i, "")
	}

	baseURL.Path += "/" + input
	params := url.Values{}
	params.Add("page", "1")
	params.Add("itemsPerPage", "1")
	baseURL.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		logger.Errorf(e, "error creating predictIt request: %s", err)
		c.Replyf(e, "unable to search PredictIt for %s", style.Bold(input))
		return
	}

	if req == nil {
		logger.Errorf(e, "nil request for predictIt")
		c.Replyf(e, "unable to search PredictIt for %s", style.Bold(input))
		return
	}

	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorf(e, "error searching predictIt: %s", err)
		c.Replyf(e, "unable to search PredictIt for %s", style.Bold(input))
		return
	}

	if resp == nil {
		logger.Errorf(e, "nil response from predictIt")
		c.Replyf(e, "unable to search PredictIt for %s", style.Bold(input))
		return
	}

	var searchResult predictItSearchResult
	if err = json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		logger.Errorf(e, "error decoding predictIt response: %s", err)
		c.Replyf(e, "unable to process PredictIt result for %s", style.Bold(input))
		return
	}

	resp.Body.Close()

	if len(searchResult.Markets) == 0 {
		logger.Debugf(e, "no markets found for %s", input)
		c.Replyf(e, "no PredictIt markets found for %s", style.Bold(input))
		return
	}

	searchResultMarket := searchResult.Markets[0]

	if len(searchResultMarket.Contracts) == 0 {
		logger.Debugf(e, "no contracts found for %s", searchResultMarket.Name)
		c.Replyf(e, "no PredictIt contracts found for %s", style.Bold(searchResultMarket.Name))
		return
	}

	resp, err = http.Get(fmt.Sprintf(predictItMarketURL, searchResultMarket.ID))
	if err != nil {
		logger.Errorf(e, "error retrieving predictIt market: %s", err)
		c.Replyf(e, "unable to retrieve PredictIt market for %s", style.Bold(searchResultMarket.Name))
		return
	}

	var marketResult predictItMarketResult
	if err = json.NewDecoder(resp.Body).Decode(&marketResult); err != nil {
		logger.Errorf(e, "error decoding predictIt market: %s", err)
		c.Replyf(e, "unable to process PredictIt market for %s", style.Bold(searchResultMarket.Name))
		return
	}

	resp.Body.Close()

	contracts := make([]string, 0)
	maxYes := 0.0
	for _, contract := range marketResult.Contracts {
		if contract.Price > maxYes {
			maxYes = contract.Price
		}
	}

	for _, contract := range marketResult.Contracts {
		message := fmt.Sprintf("%s: $%.02f", style.Underline(contract.Name), contract.Price)

		trades := ""
		if contract.Trades > 0 {
			trades = fmt.Sprintf("(%s trades)", text.DecorateNumberWithCommas(contract.Trades))
		}

		if contract.Price == maxYes {
			contracts = append(contracts, fmt.Sprintf("%s%s", style.ColorForeground(message, style.ColorGreen), trades))
		} else {
			contracts = append(contracts, fmt.Sprintf("%s%s", message, trades))
		}
	}

	detail := ""
	if searchResultMarket.Status != "Open" {
		detail = fmt.Sprintf(" (%s)", style.ColorForeground(searchResultMarket.Status, style.ColorRed))
	}

	c.SendMessages(e, e.ReplyTarget(), []string{
		fmt.Sprintf("%s%s • %s", style.Bold(searchResultMarket.Name), detail, strings.Join(contracts, " | ")),
		fmt.Sprintf(predictItMarketDetailURL, searchResultMarket.ID),
	})
}
