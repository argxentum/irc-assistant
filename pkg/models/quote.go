package models

import (
	"assistant/pkg/api/text"
	"fmt"
	"github.com/google/uuid"
	"time"
)

const quoteIDPrefix = "quote"

type Quote struct {
	ID       string    `firestore:"id"`
	Author   string    `firestore:"author"`
	Quote    string    `firestore:"quote"`
	QuotedBy string    `firestore:"quoted_by"`
	QuotedAt time.Time `firestore:"quoted_at"`
	Keywords []string  `firestore:"keywords"`
}

func NewQuote(author, quote, quotedBy string) *Quote {
	return &Quote{
		ID:       fmt.Sprintf("%s-%s", quoteIDPrefix, uuid.NewString()),
		Author:   author,
		Quote:    quote,
		QuotedBy: quotedBy,
		QuotedAt: time.Now(),
		Keywords: text.ParseKeywords(quote),
	}
}

func NewQuoteFromRecentMessage(author, quotedBy string, message RecentMessage) *Quote {
	return &Quote{
		ID:       fmt.Sprintf("%s-%s", quoteIDPrefix, uuid.NewString()),
		Author:   author,
		Quote:    message.Message,
		QuotedBy: quotedBy,
		QuotedAt: message.At,
		Keywords: text.ParseKeywords(message.Message),
	}
}
