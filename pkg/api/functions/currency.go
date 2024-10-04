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
	"math"
	"net/http"
	"strings"
	"time"
)

const currencyFunctionName = "currency"
const currencyConversionLatestURL = "https://api.freecurrencyapi.com/v1/latest?base_currency=%s&currencies=%s&apikey=%s"
const currencyConversionHistoricalURL = "https://api.freecurrencyapi.com/v1/historical?date=%s&base_currency=%s&currencies=%s&apikey=%s"
const currencyMetadataURL = "https://api.freecurrencyapi.com/v1/currencies?currencies=%s,%s&apikey=%s"

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

	metadata, err := f.currencyMetadata(from, to)
	if err != nil {
		logger.Warningf(e, "error retrieving currency metadata: %s", err)
		f.Replyf(e, "Unable to convert from %s to %s.", style.Bold(from), style.Bold(to))
		return
	}

	fromSingular := metadata.Data[from].Name
	toPlural := metadata.Data[to].NamePlural

	latest, err := f.latestConversion(from, to)
	if err != nil {
		logger.Warningf(e, "error retrieving latest currency conversion: %s", err)
		f.Replyf(e, "Unable to convert from %s to %s.", style.Bold(from), style.Bold(to))
		return
	}

	lastMonth := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	historicalMonth, err := f.historicalConversion(lastMonth, from, to)
	if err != nil {
		logger.Warningf(e, "error retrieving 1m historical currency conversion: %s", err)
		f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("1 %s (%s) = %s", text.CapitalizeEveryWord(fromSingular, false), from, style.Underline(fmt.Sprintf("%.2f %s (%s)", latest.Data[to], text.CapitalizeEveryWord(toPlural, false), to))))
		return
	}

	lastYear := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	historicalYear, err := f.historicalConversion(lastYear, from, to)
	if err != nil {
		logger.Warningf(e, "error retrieving 1y historical currency conversion: %s", err)
		f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("1 %s (%s) = %s", text.CapitalizeEveryWord(fromSingular, false), from, style.Underline(fmt.Sprintf("%.2f %s (%s)", latest.Data[to], text.CapitalizeEveryWord(toPlural, false), to))))
		return
	}

	summary := ""

	rateMonth := math.Abs(latest.Data[to]-historicalMonth.Data[lastMonth][to]) / historicalMonth.Data[lastMonth][to] * 100.0
	if historicalMonth.Data[lastMonth][to] < latest.Data[to] {
		summary = style.ColorForeground(fmt.Sprintf("▲ %.2f%%", rateMonth), style.ColorGreen) + " (1M)"
	} else {
		summary = style.ColorForeground(fmt.Sprintf("▼ %.2f%%", rateMonth), style.ColorRed) + " (1M)"
	}

	summary += " | "

	rateYear := math.Abs(latest.Data[to]-historicalYear.Data[lastYear][to]) / historicalYear.Data[lastYear][to] * 100.0
	if historicalYear.Data[lastYear][to] < latest.Data[to] {
		summary += style.ColorForeground(fmt.Sprintf("▲ %.2f%%", rateYear), style.ColorGreen) + " (1Y)"
	} else {
		summary += style.ColorForeground(fmt.Sprintf("▼ %.2f%%", rateYear), style.ColorRed) + " (1Y)"
	}

	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("1 %s (%s) = %s (%s) | %s", text.CapitalizeEveryWord(fromSingular, false), from, style.Underline(fmt.Sprintf("%.2f %s", latest.Data[to], text.CapitalizeEveryWord(toPlural, false))), to, summary))
}

type latestConversion struct {
	Data map[string]float64
}

type historicalConversion struct {
	Data map[string]map[string]float64
}

type currencyMetadata struct {
	Data map[string]struct {
		Name       string
		NamePlural string `json:"name_plural"`
	}
}

func (f *currencyFunction) latestConversion(from, to string) (latestConversion, error) {
	resp, err := http.Get(fmt.Sprintf(currencyConversionLatestURL, from, to, f.cfg.Currency.APIKey))
	if err != nil {
		return latestConversion{}, err
	}

	defer resp.Body.Close()

	var conversion latestConversion
	err = json.NewDecoder(resp.Body).Decode(&conversion)
	return conversion, err
}

func (f *currencyFunction) historicalConversion(date, from, to string) (historicalConversion, error) {
	resp, err := http.Get(fmt.Sprintf(currencyConversionHistoricalURL, date, from, to, f.cfg.Currency.APIKey))
	if err != nil {
		return historicalConversion{}, err
	}

	defer resp.Body.Close()

	var conversion historicalConversion
	err = json.NewDecoder(resp.Body).Decode(&conversion)
	return conversion, err
}

func (f *currencyFunction) currencyMetadata(from, to string) (currencyMetadata, error) {
	resp, err := http.Get(fmt.Sprintf(currencyMetadataURL, from, to, f.cfg.Currency.APIKey))
	if err != nil {
		return currencyMetadata{}, err
	}

	defer resp.Body.Close()

	var metadata currencyMetadata
	err = json.NewDecoder(resp.Body).Decode(&metadata)
	return metadata, err
}
