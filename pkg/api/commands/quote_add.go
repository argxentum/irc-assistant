package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"strings"
)

const QuoteAddCommandName = "quote_add"

type QuoteAddCommand struct {
	*commandStub
}

func NewQuoteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &QuoteAddCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *QuoteAddCommand) Name() string {
	return QuoteAddCommandName
}

func (c *QuoteAddCommand) Description() string {
	return "Saves a user's quote, searching their recent messages for any matching the given content."
}

func (c *QuoteAddCommand) Triggers() []string {
	return []string{"quote", "q"}
}

func (c *QuoteAddCommand) Usages() []string {
	return []string{"%s <nick> [<message-content>]"}
}

func (c *QuoteAddCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *QuoteAddCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *QuoteAddCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())

	tokens := Tokens(e.Message())
	nick := tokens[1]

	silent := strings.Contains(e.Raw, ".grab")
	logger.Debugf(e, "silent quote? %v", silent)

	if nick == e.From {
		if !silent {
			c.Replyf(e, "Sorry, you can't quote yourself.")
		}
		return
	}

	quote := ""
	if len(tokens) >= 3 {
		quote = strings.Join(tokens[2:], " ")
	}

	quotedBy := e.From

	user, err := repository.GetUser(e, e.ReplyTarget(), nick, false)
	if err != nil {
		logger.Errorf(e, "error getting user: %v", err)
		return
	}

	if user == nil {
		logger.Errorf(e, "user not found")
		if !silent {
			c.Replyf(e, "Sorry, I'm unable to find anyone named %s.", style.Bold(nick))
		}
		return
	}

	var msg models.RecentMessage
	ok := false
	if len(quote) == 0 {
		msg, ok = repository.FindMostRecentUserMessage(e, user)
	} else {
		msg, ok = repository.FindRecentUserMessage(e, user, quote)
	}
	if !ok {
		if !silent {
			if len(quote) == 0 {
				c.Replyf(e, "Sorry, I couldn't find any recent messages from %s.", style.Bold(nick))
			} else {
				c.Replyf(e, "Sorry, I couldn't find any recent messages from %s matching: %s", style.Bold(nick), style.Italics(quote))
			}
		}
		return
	}

	q := models.NewQuoteFromRecentMessage(nick, quotedBy, msg)
	if q == nil {
		logger.Errorf(e, "error creating quote")
		if !silent {
			c.Replyf(e, "Sorry, something went wrong while trying to save the quote.")
		}
		return
	}

	fs := firestore.Get()
	if err = fs.CreateQuote(e.ReplyTarget(), q); err != nil {
		logger.Errorf(e, "error saving quote: %v", err)
		if !silent {
			c.Replyf(e, "Sorry, I couldn't save the quote.")
		}
		return
	}

	if !silent {
		c.Replyf(e, "Saved quote: <%s> %s", style.Bold(style.Italics(q.Author)), style.Italics(q.Quote))
	}
}
