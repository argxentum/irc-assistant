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
	fmt.Printf("🔍 %s domain is %s\n", url, domain)
	if domainSpecificDirectHandling[domain] == nil {
		fmt.Printf("🔍 no domain-specific handling for %s\n", domain)
		return ""
	}

	fmt.Printf("🗒 %s requires domain-specific handling\n", url)

	message := domainSpecificDirectHandling[domain](doc)
	fmt.Printf("🔍 domain-specific message for %s: %s\n", domain, message)
	if len(message) == 0 {
		return ""
	}

	return message
}
