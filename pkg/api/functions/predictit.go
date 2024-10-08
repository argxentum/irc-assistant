package functions

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

const predictItFunctionName = "predictIt"
const predictItSearchBaseURL = "https://www.predictit.org/api/Browse/Search"
const predictItMarketDetailURL = "https://www.predictit.org/markets/detail/%d"

type predictItFunction struct {
	FunctionStub
}

func NewPredictItFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, predictItFunctionName)
	if err != nil {
		return nil, err
	}

	return &predictItFunction{
		FunctionStub: stub,
	}, nil
}

func (f *predictItFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

type predictItSearchResult struct {
	Markets []struct {
		ID        int    `json:"marketId"`
		Name      string `json:"marketName"`
		Status    string `json:"status"`
		Contracts []struct {
			ID     int     `json:"contractId"`
			Name   string  `json:"contractName"`
			Price  float64 `json:"latestTradePrice"`
			Trades int     `json:"totalTrades"`
		}
	}
}

func (f *predictItFunction) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	input := strings.Join(tokens[1:], " ")
	logger.Infof(e, "⚡ [%s/%s] predictIt %s", e.From, e.ReplyTarget(), input)

	baseURL, err := url.Parse(predictItSearchBaseURL)
	if err != nil {
		fmt.Println("Malformed URL: ", err.Error())
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
		f.Replyf(e, "unable to search PredictIt for %s", style.Bold(input))
		return
	}

	if req == nil {
		logger.Errorf(e, "nil request for predictIt")
		f.Replyf(e, "unable to search PredictIt for %s", style.Bold(input))
		return
	}

	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorf(e, "error searching predictIt: %s", err)
		f.Replyf(e, "unable to search PredictIt for %s", style.Bold(input))
		return
	}

	if resp == nil {
		logger.Errorf(e, "nil response from predictIt")
		f.Replyf(e, "unable to search PredictIt for %s", style.Bold(input))
		return
	}

	defer resp.Body.Close()

	var result predictItSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Errorf(e, "error decoding predictIt response: %s", err)
		f.Replyf(e, "unable to process PredictIt result for %s", style.Bold(input))
		return
	}

	if len(result.Markets) == 0 {
		logger.Debugf(e, "no markets found for %s", input)
		f.Replyf(e, "no PredictIt markets found for %s", style.Bold(input))
		return
	}

	market := result.Markets[0]

	contracts := make([]string, 0)
	maxYes := 0.0
	maxContractID := market.Contracts[0].ID
	for _, contract := range market.Contracts {
		if contract.Price > maxYes {
			maxYes = contract.Price
			maxContractID = contract.ID
		}
	}

	for _, contract := range market.Contracts {
		message := fmt.Sprintf("%s: $%.02f", style.Underline(contract.Name), contract.Price)
		if contract.ID == maxContractID {
			contracts = append(contracts, fmt.Sprintf("%s (%s trades)", style.ColorForeground(message, style.ColorGreen), text.DecorateNumberWithCommas(contract.Trades)))
		} else {
			contracts = append(contracts, fmt.Sprintf("%s (%s trades)", message, text.DecorateNumberWithCommas(contract.Trades)))
		}
	}

	detail := ""
	if market.Status != "Open" {
		detail = fmt.Sprintf(" (%s)", style.ColorForeground(market.Status, style.ColorRed))
	}

	f.SendMessages(e, e.ReplyTarget(), []string{
		fmt.Sprintf("%s%s • %s", style.Bold(market.Name), detail, strings.Join(contracts, " | ")),
		fmt.Sprintf(predictItMarketDetailURL, market.ID),
	})
}
