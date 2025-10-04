package commands

import (
	"assistant/pkg/api/irc"

	"github.com/bobesa/go-domain-util/domainutil"
)

var dsf map[string]func(e *irc.Event, url string) (*summary, error)

func (c *SummaryCommand) domainSummarization() map[string]func(e *irc.Event, url string) (*summary, error) {
	if dsf == nil {
		dsf = map[string]func(e *irc.Event, url string) (*summary, error){
			c.cfg.Web.Domain: c.parseShortcut,
			"youtube.com":    c.parseYouTube,
			"youtu.be":       c.parseYouTube,
			"reddit.com":     c.parseReddit,
			"twitter.com":    c.parseTwitter,
			"x.com":          c.parseTwitter,
			"fixupx.com":     c.parseTwitter,
			"fxtwitter.com":  c.parseTwitter,
			"bsky.app":       c.parseBlueSky,
			"wikipedia.org":  c.parseWikipedia,
			"polymarket.com": c.parsePolymarket,
			"kalshi.com":     c.parseKalshi,
			//"truthsocial.com": c.parseTruthSocial,
		}
	}

	return dsf
}

func (c *SummaryCommand) requiresDomainSummary(url string) bool {
	domain := domainutil.Domain(url)
	return c.domainSummarization()[domain] != nil
}

func (c *SummaryCommand) domainSummary(e *irc.Event, url string) (*summary, error) {
	domain := domainutil.Domain(url)
	if c.domainSummarization()[domain] == nil {
		return nil, nil
	}

	return c.domainSummarization()[domain](e, url)
}
