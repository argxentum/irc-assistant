package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
)

var dsf map[string]func(e *irc.Event, url string) (*summary, error)

func (c *SummaryCommand) domainSummarization() map[string]func(e *irc.Event, url string) (*summary, error) {
	if dsf == nil {
		dsf = map[string]func(e *irc.Event, url string) (*summary, error){
			"youtube.com": c.parseYouTube,
			"youtu.be":    c.parseYouTube,
			"reddit.com":  c.parseReddit,
			"twitter.com": c.parseTwitter,
			"x.com":       c.parseTwitter,
			"bsky.app":    c.parseBlueSky,
		}
	}

	return dsf
}

func (c *SummaryCommand) requiresDomainSummary(url string) bool {
	domain := retriever.RootDomain(url)
	return c.domainSummarization()[domain] != nil
}

func (c *SummaryCommand) domainSummary(e *irc.Event, url string) (*summary, error) {
	domain := retriever.RootDomain(url)
	if c.domainSummarization()[domain] == nil {
		return nil, nil
	}

	return c.domainSummarization()[domain](e, url)
}
