package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/drudge"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strconv"
	"time"
)

const DrudgeHeadlinesCommandName = "drudge_headlines"
const defaultHeadlineCount = 1
const maxHeadlineCount = 3

type DrudgeHeadlinesCommand struct {
	*commandStub
}

func NewDrudgeHeadlinesCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &DrudgeHeadlinesCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *DrudgeHeadlinesCommand) Name() string {
	return DrudgeHeadlinesCommandName
}

func (c *DrudgeHeadlinesCommand) Description() string {
	return "Shows the specified number of top headlines from Drudge Report, defaulting to 1."
}

func (c *DrudgeHeadlinesCommand) Triggers() []string {
	return []string{"drudge"}
}

func (c *DrudgeHeadlinesCommand) Usages() []string {
	return []string{"%s [<number>]"}
}

func (c *DrudgeHeadlinesCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *DrudgeHeadlinesCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *DrudgeHeadlinesCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", c.Name(), e.From, e.ReplyTarget())
	tokens := Tokens(e.Message())

	n := defaultHeadlineCount
	if len(tokens) > 1 {
		var err error
		n, err = strconv.Atoi(tokens[1])
		if err != nil {
			n = defaultHeadlineCount
		}
	}

	if n > maxHeadlineCount {
		n = maxHeadlineCount
	}

	urls, err := drudge.GetHeadlineURLs(e, n)
	if err != nil {
		logger.Warningf(e, "failed to get drudge headline URLs: %v", err)
		c.Replyf(e, "Sorry, something went wrong while trying to retrieve the latest Drudge headlines")
		return
	}

	if len(urls) == 0 {
		logger.Warningf(e, "no headline URLs found")
		c.Replyf(e, "Unable to parse any Drudge Report headlines")
	}

	go func() {
		for _, u := range urls {
			c.ExecuteSynthesizedEvent(e, SummaryCommandName, u, map[string]any{CommandMetadataShowURL: true})
			time.Sleep(3 * time.Second)
		}
	}()
}
