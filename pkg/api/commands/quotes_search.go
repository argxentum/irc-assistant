package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"regexp"
	"strings"
)

const QuotesSearchCommandName = "quotes_search"
const maxResults = 3

type QuotesSearchCommand struct {
	*commandStub
}

func NewQuotesSearchCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &QuotesSearchCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *QuotesSearchCommand) Name() string {
	return QuotesSearchCommandName
}

func (c *QuotesSearchCommand) Description() string {
	return "Searches for quotes."
}

func (c *QuotesSearchCommand) Triggers() []string {
	return []string{"quotes", "qs"}
}

func (c *QuotesSearchCommand) Usages() []string {
	return []string{"%s [by: <nick>] [<content>]"}
}

func (c *QuotesSearchCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *QuotesSearchCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

var fromRegex = regexp.MustCompile(`(?:from|by|user|nick|of|author):\s*(.*?)(?:\s|$)`)

func (c *QuotesSearchCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())

	author := ""

	raw := e.Message()
	if fromRegex.MatchString(raw) {
		fromMatch := fromRegex.FindStringSubmatch(raw)
		author = fromMatch[1]
		raw = strings.Replace(raw, fromMatch[0], "", 1)
	}

	tokens := Tokens(raw)
	content := strings.Join(tokens[1:], " ")

	keywords := make([]string, 0)
	if len(content) > 0 {
		keywords = text.ParseKeywords(content)
	}

	if len(author) == 0 && len(content) == 0 {
		c.Replyf(e, "You must provide an author and/or quote content to search for.")
		return
	}

	if len(author) == 0 && len(content) > 0 && len(keywords) == 0 {
		c.Replyf(e, "Please search again using more specific keywords.")
		return
	}

	var quotes []*models.Quote
	var err error

	if len(author) > 0 && len(keywords) > 0 {
		quotes, err = repository.FindUserQuotesWithContent(e.ReplyTarget(), author, keywords)
	} else if len(author) > 0 {
		quotes, err = repository.FindUserQuotes(e.ReplyTarget(), author)
	} else if len(keywords) > 0 {
		quotes, err = repository.FindQuotes(e.ReplyTarget(), keywords)
	}

	if err != nil {
		logger.Errorf(e, "error searching for quotes: %v", err)
		c.Replyf(e, "Sorry, I encountered an error searching for quotes.")
		return
	}

	if len(quotes) == 0 {
		if len(author) > 0 && len(content) > 0 {
			c.Replyf(e, "No quotes found for %s matching %s.", style.Bold(author), style.Bold(content))
		} else if len(author) > 0 {
			c.Replyf(e, "No quotes found for %s.", style.Bold(author))
		} else if len(content) > 0 {
			c.Replyf(e, "No quotes found matching %s.", style.Bold(content))
		} else {
			c.Replyf(e, "No quotes found.")
		}
		return
	}

	qty := "quotes"
	if len(quotes) == 1 {
		qty = "quote"
	}

	match := ""
	if len(author) > 0 && len(content) > 0 {
		match = fmt.Sprintf(" from %s matching %s", style.Bold(author), style.Bold(content))
	} else if len(author) > 0 {
		match = fmt.Sprintf(" from %s", style.Bold(author))
	} else if len(content) > 0 {
		match = fmt.Sprintf(" matching %s", style.Bold(content))
	}

	messages := make([]string, 0)
	if len(quotes) > maxResults {
		messages = append(messages, fmt.Sprintf("Found %s %s%s. Displaying most recent %s.", style.Bold(fmt.Sprintf("%d", len(quotes))), qty, match, style.Bold(fmt.Sprintf("%d", maxResults))))
		quotes = quotes[:maxResults]
	} else {
		messages = append(messages, fmt.Sprintf("Found %s %s%s.", style.Bold(fmt.Sprintf("%d", len(quotes))), qty, match))
	}

	for _, quote := range quotes {
		messages = append(messages, fmt.Sprintf("<%s> %s (%s, added by %s)", style.Bold(style.Italics(quote.Author)), style.Italics(quote.Quote), elapse.PastTimeDescription(quote.QuotedAt), quote.QuotedBy))
	}

	c.SendMessages(e, e.ReplyTarget(), messages)
}
