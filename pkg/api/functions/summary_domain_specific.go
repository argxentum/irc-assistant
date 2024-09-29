package functions

import "assistant/pkg/api/retriever"

var domainSpecificDirectHandling = map[string]func(doc string) string{
	"youtube.com": parseYouTubeMessage,
	"youtu.be":    parseYouTubeMessage,
}

func (f *summaryFunction) domainSpecificMessage(url, doc string) string {
	domain := retriever.RootDomain(url)
	if domainSpecificDirectHandling[domain] == nil {
		return ""
	}

	message := domainSpecificDirectHandling[domain](doc)
	if len(message) == 0 {
		return ""
	}

	return message
}
