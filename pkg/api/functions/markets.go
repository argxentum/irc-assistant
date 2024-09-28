package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"math/rand/v2"
	"net/url"
	"strings"
	"time"
	"unicode"
)

const marketsFunctionName = "markets"
const maxMarketAttempts = 5

type marketsFunction struct {
	FunctionStub
}

func NewMarketsFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, marketsFunctionName)
	if err != nil {
		return nil, err
	}

	return &marketsFunction{
		FunctionStub: stub,
	}, nil
}

func (f *marketsFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 0)
}

func (f *marketsFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	region := "US"
	if len(tokens) > 1 {
		region = tokens[1]
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] markets %s", e.From, e.ReplyTarget(), region)

	message := ""
	attempts := 1

	go func() {
		for {
			if attempts > maxMarketAttempts {
				break
			}

			logger.Debugf(e, "attempt %d to retrieve market summary data for %s", attempts, region)

			message = f.retrieveMarketSummaryMessage(e, region)
			if len(message) > 0 {
				f.SendMessage(e, e.ReplyTarget(), message)
				return
			}

			attempts++
			time.Sleep(time.Duration(200 + rand.IntN(150)))
		}

		f.Replyf(e, "unable to retrieve %s market data.", style.Bold(region))
	}()
}

func (f *marketsFunction) retrieveMarketSummaryMessage(e *irc.Event, region string) string {
	logger := log.Logger()

	query := url.QueryEscape(fmt.Sprintf("stock markets %s", region))
	doc, err := f.getDocument(e, fmt.Sprintf(bingSearchURL, query), true)
	if err != nil {
		logger.Warningf(e, "unable to retrieve %s stock market information: %s", region, err)
		return ""
	}

	section := doc.Find("div.finmkt").First()
	title := strings.ToLower(strings.TrimSpace(section.Find("h2").First().Text()))

	market := strings.TrimSpace(strings.Replace(strings.ToLower(strings.TrimSpace(section.Find("li").First().Text())), "market", "", -1))
	if len(market) > 0 {
		market = string(unicode.ToUpper(rune(market[0]))) + strings.ToLower(market[1:])
		caps := []string{"us", "uk", "eu", "as", "au", "ca", "nz"}
		for _, c := range caps {
			if strings.ToLower(market) == c {
				market = strings.ToUpper(market)
				break
			}
		}
		title = fmt.Sprintf("%s %s", market, title)
	}

	message := ""

	markets := section.Find("div.finind_ind").First()
	markets.Find("div.finind_item").Each(func(i int, s *goquery.Selection) {
		ticker := strings.TrimSpace(s.Find("div.finind_ticker").First().Text())
		val := s.Find("div.finind_val").First()
		value := strings.TrimSpace(val.Text())
		change := strings.TrimSpace(val.Next().Text())

		if len(ticker) == 0 || len(value) == 0 {
			logger.Warningf(e, "skipping invalid stock market information: %s %s %s", ticker, value, change)
			return
		}

		styledChange := change
		if strings.HasPrefix(change, "▼") {
			styledChange = style.ColorForeground(change, style.ColorRed)
		} else if strings.HasPrefix(change, "▲") {
			styledChange = style.ColorForeground(change, style.ColorGreen)
		}

		if len(message) > 0 {
			message += " | "
		}

		message += fmt.Sprintf("%s: %s %s", style.Underline(ticker), value, styledChange)
	})

	if len(message) == 0 {
		logger.Warningf(e, "no stock market information found")
		return ""
	}

	message = style.Bold(title) + " – " + message
	return message
}
