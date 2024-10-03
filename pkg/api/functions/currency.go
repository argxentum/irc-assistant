package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"github.com/go-viper/mapstructure/v2"
	"math"
	"net/http"
	"net/url"
	"strings"
)

const currencyFunctionName = "currency"
const currencyConversionURL = "https://www.xe.com/api/protected/midmarket-converter"
const currencyConversionStatisticsURL = "https://www.xe.com/api/protected/statistics"
const currencyConversionDetailsURL = "https://www.xe.com/currencyconverter/convert/?Amount=1&From=%s&To=%s"

type currencyFunction struct {
	FunctionStub
}

func NewCurrencyFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, currencyFunctionName)
	if err != nil {
		return nil, err
	}

	return &currencyFunction{
		FunctionStub: stub,
	}, nil
}

func (f *currencyFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *currencyFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	msg := e.Message()
	msg = strings.ReplaceAll(msg, " to ", " ")

	tokens := Tokens(e.Message())
	from := "USD"
	to := ""

	if len(tokens) < 3 {
		to = tokens[1]
	} else {
		from = tokens[1]
		to = tokens[2]
	}

	from = strings.ToUpper(from)
	to = strings.ToUpper(to)

	log.Logger().Infof(e, "⚡ [%s/%s] currency %s to %s", e.From, e.ReplyTarget(), from, to)

	conv, err := f.convertCurrencies(from, to)
	if err != nil {
		logger.Warningf(e, "error converting currencies: %s", err)
		f.Replyf(e, "Unable to convert from %s to %s.", style.Bold(from), style.Bold(to))
		return
	}

	stats, err := f.currencyStatistics(from, to)
	if err != nil {
		logger.Warningf(e, "error retrieving currency statistics: %s", err)

		f.SendMessages(e, e.ReplyTarget(), []string{
			fmt.Sprintf("1 %s = %s", conv.From, style.Underline(fmt.Sprintf("%.2f %s", conv.Rate, conv.To))),
			fmt.Sprintf(currencyConversionDetailsURL, conv.From, conv.To),
		})
		return
	}

	summary90days := ""
	rate := math.Abs(conv.Rate-stats.Last90Days.Average) / conv.Rate * 100.0
	if stats.Last90Days.Average < conv.Rate {
		summary90days = style.ColorForeground(fmt.Sprintf("▼ %.2f%%", rate), style.ColorRed)
	} else {
		summary90days = style.ColorForeground(fmt.Sprintf("▲ %.2f%%", rate), style.ColorGreen)
	}

	f.SendMessages(e, e.ReplyTarget(), []string{
		fmt.Sprintf("1 %s = %s (%s, 90 days)", conv.From, style.Underline(fmt.Sprintf("%.2f %s", conv.Rate, conv.To)), summary90days),
		fmt.Sprintf(currencyConversionDetailsURL, conv.From, conv.To),
	})
}

type usdConversionRates struct {
	Timestamp float64
	Rates     map[string]float64
}

type conversionStatistics struct {
	Last1Days  conversionStatistic `mapstructure:"last1days"`
	Last7Days  conversionStatistic `mapstructure:"last7days"`
	Last30Days conversionStatistic `mapstructure:"last30days"`
	Last60Days conversionStatistic `mapstructure:"last60days"`
	Last90Days conversionStatistic `mapstructure:"last90days"`
}

type conversionStatistic struct {
	To                string
	High              float64
	Low               float64
	Average           float64
	StandardDeviation float64
	Volatility        float64
}

type conversion struct {
	From string
	To   string
	Rate float64
}

func (f *currencyFunction) convertCurrencies(from, to string) (conversion, error) {
	req, err := http.NewRequest(http.MethodGet, currencyConversionURL, nil)
	if err != nil {
		return conversion{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", f.cfg.XE.APIKey))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return conversion{}, err
	}

	var usdRates usdConversionRates
	if err = json.NewDecoder(resp.Body).Decode(&usdRates); err != nil {
		return conversion{}, err
	}

	fromRate, ok := usdRates.Rates[from]
	if !ok {
		return conversion{}, fmt.Errorf("no rate for %s", from)
	}

	toRate, ok := usdRates.Rates[to]
	if !ok {
		return conversion{}, fmt.Errorf("no rate for %s", to)
	}

	return conversion{
		From: from,
		To:   to,
		Rate: (1 / fromRate) * toRate,
	}, nil
}

func (f *currencyFunction) currencyStatistics(from, to string) (conversionStatistics, error) {
	req, err := http.NewRequest(http.MethodGet, currencyConversionStatisticsURL, nil)
	if err != nil {
		return conversionStatistics{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", f.cfg.XE.APIKey))
	req.URL.RawQuery = url.Values{
		"from": {from},
		"to":   {to},
	}.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return conversionStatistics{}, err
	}

	var stats map[string]map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return conversionStatistics{}, err
	}

	statistics := conversionStatistics{}
	_ = mapstructure.Decode(stats, &statistics)
	return statistics, nil
}
