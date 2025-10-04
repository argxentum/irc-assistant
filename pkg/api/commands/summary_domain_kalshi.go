package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/kalshi"
	"assistant/pkg/log"
	"fmt"
	"regexp"
)

var kalshiURLPattern = regexp.MustCompile("^https://kalshi.com/markets/.*?/(?:.*?/)?(.*?)$")

func (c *SummaryCommand) parseKalshi(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()

	m := kalshiURLPattern.FindStringSubmatch(url)
	if m == nil || len(m) < 2 {
		return nil, fmt.Errorf("invalid kalshi url: %s", url)
	}

	eventTicker := m[1]
	logger.Debugf(e, "Parse kalshi event ticker %s from %s", eventTicker, url)

	event, markets, err := kalshi.GetEventAndMarkets(eventTicker)
	if err != nil {
		return nil, fmt.Errorf("error getting kalshi events and markets: %w", err)
	}

	logger.Debugf(e, "Retrieved %d markets for event %s (%s)", len(markets), event.EventTicker, event.SeriesTicker)
	messages := kalshi.Summarize(event, markets, false)
	return createSummary(messages...), nil
}
