package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
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
	return "Searches for a random quote from the specified user."
}

func (c *QuoteRandomCommand) Triggers() []string {
	return []string{"quoter", "qr"}
}

func (c *QuoteRandomCommand) Usages() []string {
	return []string{"%s <nick>"}
}

func (c *QuoteRandomCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *QuoteRandomCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *QuoteRandomCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())

	tokens := Tokens(e.Message())
	nick := tokens[1]

	quotes, err := repository.FindUserQuotes(e.ReplyTarget(), nick)
	if err != nil {
		c.Replyf(e, "Unable to find quotes from %s", style.Bold(nick))
		return
	}

	if len(quotes) == 0 {
		c.Replyf(e, "No quotes found from %s", style.Bold(nick))
		return
	}

	quote := quotes[rand.Intn(len(quotes))]

	messages := []string{
		"Random quote from " + style.Bold(nick) + ":",
		fmt.Sprintf("<%s> %s (%s, added by %s)", style.Bold(style.Italics(quote.Author)), style.Italics(quote.Quote), elapse.PastTimeDescription(quote.QuotedAt), quote.QuotedBy),
	}

	c.SendMessages(e, e.ReplyTarget(), messages)
}
