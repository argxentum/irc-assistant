package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/kalshi"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const KalshiCommandName = "kalshi"

type KalshiCommand struct {
	*commandStub
}

func NewKalshiCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &KalshiCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *KalshiCommand) Name() string {
	return KalshiCommandName
}

func (c *KalshiCommand) Description() string {
	return "Displays the latest Kalshi betting data for the market matching the query."
}

func (c *KalshiCommand) Triggers() []string {
	return []string{"kalshi"}
}

func (c *KalshiCommand) Usages() []string {
	return []string{"%s <query>"}
}

func (c *KalshiCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *KalshiCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *KalshiCommand) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	query := strings.Join(tokens[1:], " ")
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), query)

	event, err := kalshi.FindEvent(query, "", 0)
	if err != nil {
		logger.Errorf(e, "error fetching Kalshi event results: %s", err)
		c.Replyf(e, "Error fetching Kalshi events data")
		return
	}

	if event == nil {
		logger.Warningf(e, "no Kalshi event results found for query: %s", query)
		c.Replyf(e, "No active Kalshi results found for %s", style.Bold(query))
		return
	}

	logger.Infof(e, "found Kalshi event: %s", event.EventTicker)

	markets, err := kalshi.FindMarkets(event.EventTicker)
	if err != nil {
		logger.Errorf(e, "error fetching Kalshi markets: %s", err)
		c.Replyf(e, "Error fetching Kalshi markets data")
		return
	}

	if len(markets) == 0 {
		logger.Debugf(e, "no Kalshi markets found for %s", query)
		c.Replyf(e, "No active Kalshi markets found for %s (%s)", style.Bold(query), style.Italics(event.EventTicker))
		return
	}

	messages := kalshi.Summarize(event, markets, true)

	if messages == nil {
		c.Replyf(e, "No Kalshi markets found for %s", style.Bold(query))
		return
	}

	c.irc.SendMessages(e.ReplyTarget(), messages)
}
