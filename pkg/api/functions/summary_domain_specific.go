package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
)

type summary struct {
	text string
}

var dsf map[string]func(e *irc.Event, url string) (*summary, error)

func (f *summaryFunction) domainSpecificSummarization() map[string]func(e *irc.Event, url string) (*summary, error) {
	if dsf == nil {
		dsf = map[string]func(e *irc.Event, url string) (*summary, error){
			"youtube.com": f.parseYouTube,
			"youtu.be":    f.parseYouTube,
			"reddit.com":  f.parseRedditMessage,
		}
	}

	return dsf
}

func (f *summaryFunction) requiresDomainSpecificSummarization(url string) bool {
	domain := retriever.RootDomain(url)
	return f.domainSpecificSummarization()[domain] != nil
}

func (f *summaryFunction) domainSpecificSummary(e *irc.Event, url string) (*summary, error) {
	domain := retriever.RootDomain(url)
	if f.domainSpecificSummarization()[domain] == nil {
		return nil, nil
	}

	return f.domainSpecificSummarization()[domain](e, url)
}
