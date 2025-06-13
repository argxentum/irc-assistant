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
	"strings"
	"time"
)

const GoogleFinanceMarketsCommandName = "markets_google_finance"
const cacheTimeout = 1 * time.Minute

type GoogleFinanceMarketsCommand struct {
	*commandStub
	cache     map[int]string
	retriever retriever.DocumentRetriever
}

func NewGoogleFinanceMarketsCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &GoogleFinanceMarketsCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		cache:       make(map[int]string),
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *GoogleFinanceMarketsCommand) Name() string {
	return GoogleFinanceMarketsCommandName
}

func (c *GoogleFinanceMarketsCommand) Description() string {
	return "Displays current stock market data."
}

func (c *GoogleFinanceMarketsCommand) Triggers() []string {
	return []string{"markets", "market", "stocks"}
}

func (c *GoogleFinanceMarketsCommand) Usages() []string {
	return []string{"%s"}
}

func (c *GoogleFinanceMarketsCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *GoogleFinanceMarketsCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *GoogleFinanceMarketsCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", c.Name(), e.From, e.ReplyTarget())

	for timestamp, message := range c.cache {
		if time.Since(time.Unix(int64(timestamp), 0)) < cacheTimeout {
			c.SendMessage(e, e.ReplyTarget(), message)
			logger.Debugf(e, "using cached stock market data")
			return
		}
	}

	message := c.retrieveSummary(e)
	if len(message) == 0 {
		logger.Warning(e, "unable to retrieve stock market data")
		c.Replyf(e, "Unable to retrieve stock market data")
		return
	}

	currentTimestamp := int(time.Now().Unix())
	c.cache = make(map[int]string)
	c.cache[currentTimestamp] = message

	c.SendMessage(e, e.ReplyTarget(), message)
}

func (c *GoogleFinanceMarketsCommand) retrieveSummary(e *irc.Event) string {
	logger := log.Logger()

	r := retriever.NewDocumentRetriever(retriever.NewBodyRetriever())
	doc, err := r.RetrieveDocument(e, retriever.DefaultParams("https://finance.google.com"))
	if err != nil {
		logger.Errorf(e, "error retrieving stock market data: %s", err)
		return ""
	}

	message := ""

	doc.Root.Find(`c-wiz div[role='complementary'] a[href^='./quote/']`).Each(func(i int, s *goquery.Selection) {
		firstColumn := s.Find("a>div>div").First()
		nameNode := firstColumn.Find("div").First()
		name := strings.TrimSpace(nameNode.Text())
		price := strings.TrimSpace(nameNode.Siblings().First().Text())
		secondColumn := firstColumn.Next()
		changePercentNode := secondColumn.Find("span").First()
		changePercent := strings.TrimSpace(changePercentNode.Text())
		change := strings.TrimSpace(changePercentNode.Siblings().First().Text())

		if len(name) == 0 || len(price) == 0 {
			logger.Debugf(e, "skipping invalid stock market entry: %s", s.Text())
			return
		}

		styledChange := ""
		if strings.HasPrefix(change, "-") {
			if len(changePercent) > 0 {
				styledChange = style.ColorForeground(fmt.Sprintf("▼ %s (%s)", change, changePercent), style.ColorRed)
			} else {
				styledChange = style.ColorForeground(fmt.Sprintf("▼ %s", change), style.ColorRed)
			}
		} else if len(change) > 0 {
			if len(changePercent) > 0 {
				styledChange = style.ColorForeground(fmt.Sprintf("▲ %s (%s)", change, changePercent), style.ColorGreen)
			} else {
				styledChange = style.ColorForeground(fmt.Sprintf("▲ %s", change), style.ColorGreen)
			}
		}

		if len(styledChange) == 0 {
			styledChange = "N/A"
		}

		if len(message) > 0 {
			message += " | "
		}

		message += fmt.Sprintf("%s: %s %s", style.Bold(name), style.Underline(price), styledChange)
	})

	return message
}
