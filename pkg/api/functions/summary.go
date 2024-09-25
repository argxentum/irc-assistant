package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const summaryFunctionName = "summary"

var allowedContentTypePrefixes = []string{
	"text/html",
	"text/plain",
	"text/xml",
	"application/xml",
	"application/xhtml",
	"application/rss",
	"application/atom",
	"application/rdf",
	"application/json",
	"application/ld+json",
	"application/vnd.api",
	"application/hal+json",
	"application/vnd.collection",
}

type summaryFunction struct {
	FunctionStub
}

func NewSummaryFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, summaryFunctionName)
	if err != nil {
		return nil, err
	}

	return &summaryFunction{
		FunctionStub: stub,
	}, nil
}

func (f *summaryFunction) MayExecute(e *irc.Event) bool {
	if !f.isValid(e, 0) {
		return false
	}

	tokens := Tokens(e.Message())
	return strings.HasPrefix(tokens[0], "https://") || strings.HasPrefix(tokens[0], "http://")
}

func (f *summaryFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	url := translateURL(tokens[0])

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] summary %s", e.From, e.ReplyTarget(), url)

	f.tryDirect(e, url, false)
}

const minimumTitleLength = 16
const minimumPreferredTitleLength = 64
const maximumPreferredTitleLength = 128

var descriptionDomainDenylist = []string{
	"imgur.com",
	"youtube.com",
	"youtu.be",
}

func (f *summaryFunction) tryDirect(e *irc.Event, url string, impersonated bool) {
	logger := log.Logger()
	logger.Infof(e, "trying direct (impersonated: %t) for %s", impersonated, url)

	doc, err := getDocument(url, impersonated)
	if errors.Is(err, disallowedContentTypeError) {
		return
	}
	if err != nil || doc == nil {
		logger.Debugf(e, "unable to retrieve %s (impersonated: %t): %s", url, impersonated, err)
		if !impersonated {
			f.tryDirect(e, url, true)
			return
		}
		f.tryNuggetize(e, url)
		return
	}

	domainSpecific := f.domainSpecificMessage(url, doc.Text())
	if len(domainSpecific) > 0 {
		logger.Debugf(e, "performed domain specific handling: %s", url)
		f.SendMessage(e, e.ReplyTarget(), domainSpecific)
		return
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())
	titleAttr, _ := doc.Find("meta[property='og:title']").First().Attr("content")
	titleMeta := strings.TrimSpace(titleAttr)
	descriptionAttr, _ := doc.Find("html meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descriptionAttr)
	h1 := strings.TrimSpace(doc.Find("html body h1").First().Text())

	if len(titleAttr) > 0 {
		title = titleMeta
	} else if len(h1) > 0 {
		title = h1
	}

	if len(title)+len(description) < minimumTitleLength {
		logger.Debugf(e, "title and description too short, title: %s, description: %s", title, description)
		f.tryNuggetize(e, url)
		return
	}

	includeDescription := true
	if isDomainDenylisted(url, descriptionDomainDenylist) {
		includeDescription = false
	}

	logger.Debugf(e, "title: %s, description: %s", title, description)

	if includeDescription && len(title) > 0 && len(description) > 0 && (len(title)+len(description) < maximumPreferredTitleLength || len(title) < minimumPreferredTitleLength) {
		if strings.Contains(description, title) || strings.Contains(title, description) {
			if len(description) > len(title) {
				f.SendMessage(e, e.ReplyTarget(), style.Bold(description))
				return
			}
			f.SendMessage(e, e.ReplyTarget(), style.Bold(title))
			return
		}
		f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s: %s", style.Bold(title), description))
		return
	}
	if len(title) > 0 {
		f.SendMessage(e, e.ReplyTarget(), style.Bold(title))
		return
	}
	if includeDescription && len(description) > 0 {
		f.SendMessage(e, e.ReplyTarget(), style.Bold(description))
		return
	}

	if !impersonated {
		f.tryDirect(e, url, true)
	} else {
		f.tryNuggetize(e, url)
	}
}

func (f *summaryFunction) tryNuggetize(e *irc.Event, url string) {
	logger := log.Logger()
	logger.Infof(e, "trying nuggetize for %s", url)

	doc, err := getDocument(fmt.Sprintf("https://nug.zip/%s", url), true)
	if err != nil || doc == nil {
		logger.Debugf(e, "unable to retrieve nuggetize summary for %s: %s", url, err)
		f.tryBing(e, url)
		return
	}

	title := strings.TrimSpace(doc.Find("span.title").First().Text())

	if len(title) < minimumTitleLength {
		logger.Debugf(e, "nuggetize title too short: %s", title)
		f.tryBing(e, url)
		return
	} else {
		f.SendMessage(e, e.ReplyTarget(), style.Bold(title))
		return
	}
}

var bingDomainDenylist = []string{
	"youtube.com",
	"youtu.be",
	"twitter.com",
	"x.com",
}

func (f *summaryFunction) tryBing(e *irc.Event, url string) {
	logger := log.Logger()
	logger.Infof(e, "trying bing for %s", url)

	if isDomainDenylisted(url, bingDomainDenylist) {
		logger.Debugf(e, "bing domain denylisted %s", url)
		f.tryDuckDuckGo(e, url)
		return
	}

	doc, err := getDocument(fmt.Sprintf(bingSearchURL, url), true)
	if err != nil || doc == nil {
		logger.Debugf(e, "unable to retrieve bing search results for %s: %s", url, err)
		f.tryDuckDuckGo(e, url)
		return
	}

	title := strings.TrimSpace(doc.Find("ol#b_results").First().Find("h2").First().Text())

	if strings.Contains(strings.ToLower(title), fmt.Sprintf("about %s", url[:min(len(url), 24)])) {
		logger.Debugf(e, "bing title contains 'about <url>': %s", title)
		f.tryDuckDuckGo(e, url)
		return
	}

	if len(title) > 0 {
		f.SendMessage(e, e.ReplyTarget(), style.Bold(title))
		return
	}

	f.tryDuckDuckGo(e, url)
}

var duckDuckGoDomainDenylist = []string{
	//
}

func (f *summaryFunction) tryDuckDuckGo(e *irc.Event, url string) {
	logger := log.Logger()
	logger.Infof(e, "trying duckduckgo for %s", url)

	if isDomainDenylisted(url, duckDuckGoDomainDenylist) {
		logger.Debugf(e, "duckduckgo domain denylisted %s", url)
		return
	}

	doc, err := getDocument(fmt.Sprintf("https://html.duckduckgo.com/html?q=%s", url), true)
	if err != nil || doc == nil {
		logger.Debugf(e, "unable to retrieve duckduckgo search results for %s: %s", url, err)
		return
	}

	title := strings.TrimSpace(doc.Find("div.result__body").First().Find("h2.result__title").First().Text())

	if strings.Contains(strings.ToLower(title), fmt.Sprintf("about %s", url[:min(len(url), 24)])) {
		logger.Debugf(e, "duckduckgo title contains 'about <url>': %s", title)
		return
	}

	if len(title) > 0 {
		f.SendMessage(e, e.ReplyTarget(), style.Bold(title))
		return
	}

	logger.Debugf(e, "unable to summarize %s", url)
}

func isDomainDenylisted(url string, denylist []string) bool {
	root := rootDomain(url)
	return slices.Contains(denylist, root)
}
