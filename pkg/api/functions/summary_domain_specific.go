package functions

import (
	"assistant/pkg/api/retriever"
	"github.com/PuerkitoBio/goquery"
)

var domainSpecificDirectHandling = map[string]func(url string, doc *goquery.Document) string{
	"youtube.com": parseYouTubeMessage,
	"youtu.be":    parseYouTubeMessage,
	"reddit.com":  parseRedditMessage,
}

func (f *summaryFunction) domainSpecificMessage(url string, doc *goquery.Document) string {
	domain := retriever.RootDomain(url)
	if domainSpecificDirectHandling[domain] == nil {
		return ""
	}

	message := domainSpecificDirectHandling[domain](url, doc)
	if len(message) == 0 {
		return ""
	}

	return message
}
