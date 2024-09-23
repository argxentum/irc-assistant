package functions

import (
	"assistant/pkg/api/core"
)

var domainSpecificDirectHandling = map[string]func(doc string) string{
	"youtube.com": parseYoutube,
	"youtu.be":    parseYoutube,
}

func (f *summaryFunction) handleDomainSpecific(e *core.Event, url, doc string) bool {
	domain := rootDomain(url)
	if domainSpecificDirectHandling[domain] == nil {
		return false
	}

	message := domainSpecificDirectHandling[domain](doc)
	if len(message) == 0 {
		return false
	}

	f.irc.SendMessage(e.ReplyTarget(), message)
	return true
}
