package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"math/rand"
)

const QuoteRandomCommandName = "random_quote"

type QuoteRandomCommand struct {
	*commandStub
}

func NewQuoteRandomCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &QuoteRandomCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *QuoteRandomCommand) Name() string {
	return QuoteRandomCommandName
}

func (c *QuoteRandomCommand) Description() string {
	return "Searches for a random quote from the specified user. If no user is specified, it will search for a random quote from the current channel."
}

func (c *QuoteRandomCommand) Triggers() []string {
	return []string{"quoter", "qr"}
}

func (c *QuoteRandomCommand) Usages() []string {
	return []string{"%s [<nick>]"}
}

func (c *QuoteRandomCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *QuoteRandomCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *QuoteRandomCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	tokens := Tokens(e.Message())

	nick := ""
	if len(tokens) > 1 {
		nick = tokens[1]
	}

	quotes := make([]*models.Quote, 0)
	var err error
	if nick == "" {
		logger.Debugf(e, "Searching for random quote in channel %s", e.ReplyTarget())
		quotes, err = repository.FindChannelQuotes(e.ReplyTarget())
	} else {
		logger.Debugf(e, "Searching for random quote from user %s in channel %s", nick, e.ReplyTarget())
		quotes, err = repository.FindUserQuotes(e.ReplyTarget(), nick)
		if err != nil {
			c.Replyf(e, "Unable to find quotes from %s", style.Bold(nick))
			return
		}
	}

	if len(quotes) == 0 {
		c.Replyf(e, "No quotes found from %s", style.Bold(nick))
		return
	}

	quote := quotes[rand.Intn(len(quotes))]

	preamble := fmt.Sprintf("Random quote:")
	if nick != "" {
		preamble = fmt.Sprintf("Random %s quote:", style.Bold(nick))
	}

	messages := []string{
		preamble,
		fmt.Sprintf("<%s> %s (%s, added by %s)", style.Bold(style.Italics(quote.Author)), style.Italics(quote.Quote), elapse.PastTimeDescription(quote.QuotedAt), quote.QuotedBy),
	}

	c.SendMessages(e, e.ReplyTarget(), messages)
}
