package functions

import (
	"fmt"
)

var domainSpecificDirectHandling = map[string]func(doc string) string{
	"youtube.com": parseYouTubeMessage,
	"youtu.be":    parseYouTubeMessage,
}

func (f *summaryFunction) domainSpecificMessage(url, doc string) string {
	domain := rootDomain(url)
	if domainSpecificDirectHandling[domain] == nil {
		return ""
	}

	fmt.Printf("🗒 %s requires domain-specific handling\n", url)

	message := domainSpecificDirectHandling[domain](doc)
	if len(message) == 0 {
		return ""
	}

	return message
}
