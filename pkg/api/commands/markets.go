package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/url"
	"strings"
	"unicode"
)

const MarketsCommandName = "markets"

type MarketsCommand struct {
	*commandStub
	retriever retriever.DocumentRetriever
}

func NewMarketsCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &MarketsCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *MarketsCommand) Name() string {
	return MarketsCommandName
}

func (c *MarketsCommand) Description() string {
	return "Displays current stock market data for the given region. Defaults to US."
}

func (c *MarketsCommand) Triggers() []string {
	return []string{"markets", "market"}
}

func (c *MarketsCommand) Usages() []string {
	return []string{"%s", "%s <region>"}
}

func (c *MarketsCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *MarketsCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *MarketsCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	region := "US"
	if len(tokens) > 1 {
		region = tokens[1]
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), region)

	message := c.retrieveMarketSummaryMessage(e, region)
	if len(message) == 0 {
		logger.Warningf(e, "unable to retrieve stock market information for %s", region)
		c.Replyf(e, "Unable to retrieve stock market information for %s", region)
		return
	}

	c.SendMessage(e, e.ReplyTarget(), message)
}

func (c *MarketsCommand) retrieveMarketSummaryMessage(e *irc.Event, region string) string {
	logger := log.Logger()

	query := url.QueryEscape(fmt.Sprintf("stock markets %s", region))
	node, err := c.retriever.RetrieveDocumentSelection(e, retriever.DefaultParams(fmt.Sprintf(bingSearchURL, query)), "div.finmkt")
	if err != nil {
		logger.Warningf(e, "unable to retrieve %s stock market information: %s", region, err)
		return ""
	}

	title := strings.ToLower(strings.TrimSpace(node.Find("h2").First().Text()))

	market := strings.TrimSpace(strings.Replace(strings.ToLower(strings.TrimSpace(node.Find("li").First().Text())), "market", "", -1))
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

	markets := node.Find("div.finind_ind").First()
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
