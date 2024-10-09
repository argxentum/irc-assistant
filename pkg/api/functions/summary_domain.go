package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
)

var dsf map[string]func(e *irc.Event, url string) (*summary, error)

func (f *summaryFunction) domainSummarization() map[string]func(e *irc.Event, url string) (*summary, error) {
	if dsf == nil {
		dsf = map[string]func(e *irc.Event, url string) (*summary, error){
			"youtube.com": f.parseYouTube,
			"youtu.be":    f.parseYouTube,
			"reddit.com":  f.parseReddit,
			"twitter.com": f.parseTwitter,
			"x.com":       f.parseTwitter,
			"bsky.app":    f.parseBlueSky,
		}
	}

	return dsf
}

func (f *summaryFunction) requiresDomainSummary(url string) bool {
	domain := retriever.RootDomain(url)
	return f.domainSummarization()[domain] != nil
}

func (f *summaryFunction) domainSummary(e *irc.Event, url string) (*summary, error) {
	domain := retriever.RootDomain(url)
	if f.domainSummarization()[domain] == nil {
		return nil, nil
	}

	return f.domainSummarization()[domain](e, url)
}
