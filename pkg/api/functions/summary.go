package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
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

func NewSummaryFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, summaryFunctionName)
	if err != nil {
		return nil, err
	}

	return &summaryFunction{
		FunctionStub: stub,
	}, nil
}

func (f *summaryFunction) MayExecute(e *core.Event) bool {
	if !f.isValid(e, 0) {
		return false
	}

	tokens := Tokens(e.Message())
	return strings.HasPrefix(tokens[0], "https://") || strings.HasPrefix(tokens[0], "http://")
}

func (f *summaryFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ summary\n")
	tokens := Tokens(e.Message())
	url := translateURL(tokens[0])
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

func (f *summaryFunction) tryDirect(e *core.Event, url string, impersonated bool) {
	fmt.Printf("🗒 trying direct (impersonated: %v) for %s\n", impersonated, url)

	doc, err := getDocument(url, impersonated)
	if errors.Is(err, disallowedContentTypeError) {
		return
	}
	if err != nil || doc == nil {
		if !impersonated {
			f.tryDirect(e, url, true)
			return
		}
		f.tryNuggetize(e, url)
		return
	}

	domainSpecific := f.domainSpecificMessage(url, doc.Text())
	if len(domainSpecific) > 0 {
		f.irc.SendMessage(e.ReplyTarget(), domainSpecific)
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
		f.tryNuggetize(e, url)
		return
	}

	includeDescription := true
	if isDomainDenylisted(url, descriptionDomainDenylist) {
		includeDescription = false
	}

	if includeDescription && len(title) > 0 && len(description) > 0 && (len(title)+len(description) < maximumPreferredTitleLength || len(title) < minimumPreferredTitleLength) {
		if strings.Contains(description, title) || strings.Contains(title, description) {
			if len(description) > len(title) {
				f.irc.SendMessage(e.ReplyTarget(), text.Bold(description))
				return
			}
			f.irc.SendMessage(e.ReplyTarget(), text.Bold(title))
			return
		}
		f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s: %s", text.Bold(title), description))
		return
	}
	if len(title) > 0 {
		f.irc.SendMessage(e.ReplyTarget(), text.Bold(title))
		return
	}
	if includeDescription && len(description) > 0 {
		f.irc.SendMessage(e.ReplyTarget(), text.Bold(description))
		return
	}

	if !impersonated {
		f.tryDirect(e, url, true)
	} else {
		f.tryNuggetize(e, url)
	}
}

func (f *summaryFunction) tryNuggetize(e *core.Event, url string) {
	fmt.Printf("🗒 trying nuggetize for %s\n", url)

	doc, err := getDocument(fmt.Sprintf("https://nug.zip/%s", url), true)
	if err != nil || doc == nil {
		f.tryBing(e, url)
		return
	}

	title := strings.TrimSpace(doc.Find("span.title").First().Text())

	if len(title) < minimumTitleLength {
		f.tryBing(e, url)
		return
	} else {
		f.irc.SendMessage(e.ReplyTarget(), text.Bold(title))
		return
	}
}

var bingDomainDenylist = []string{
	"youtube.com",
	"youtu.be",
	"twitter.com",
	"x.com",
}

func (f *summaryFunction) tryBing(e *core.Event, url string) {
	if isDomainDenylisted(url, bingDomainDenylist) {
		fmt.Printf("⚠️ summarization failed, bing domain denylisted %s\n", url)
		f.tryDuckDuckGo(e, url)
		return
	}

	fmt.Printf("🗒 trying bing for %s\n", url)

	doc, err := getDocument(fmt.Sprintf(bingSearchURL, url), true)
	if err != nil || doc == nil {
		f.tryDuckDuckGo(e, url)
		return
	}

	title := strings.TrimSpace(doc.Find("ol#b_results").First().Find("h2").First().Text())

	if strings.Contains(strings.ToLower(title), fmt.Sprintf("about %s", url[:min(len(url), 24)])) {
		f.tryDuckDuckGo(e, url)
		return
	}

	if len(title) > 0 {
		f.irc.SendMessage(e.ReplyTarget(), text.Bold(title))
		return
	}

	f.tryDuckDuckGo(e, url)
}

var duckDuckGoDomainDenylist = []string{
	//
}

func (f *summaryFunction) tryDuckDuckGo(e *core.Event, url string) {
	if isDomainDenylisted(url, duckDuckGoDomainDenylist) {
		fmt.Printf("⚠️ summarization failed, duckduckgo domain denylisted %s\n", url)
		return
	}

	fmt.Printf("🗒 trying duckduckgo for %s\n", url)

	doc, err := getDocument(fmt.Sprintf("https://html.duckduckgo.com/html?q=%s", url), true)
	if err != nil || doc == nil {
		fmt.Printf("⚠️ summarization failed, error retrieving %s\n", url)
		return
	}

	title := strings.TrimSpace(doc.Find("div.result__body").First().Find("h2.result__title").First().Text())

	if strings.Contains(strings.ToLower(title), fmt.Sprintf("about %s", url[:min(len(url), 24)])) {
		fmt.Printf("⚠️ summarization failed, title is about URL %s\n", url)
		return
	}

	if len(title) > 0 {
		f.irc.SendMessage(e.ReplyTarget(), text.Bold(title))
		return
	}

	fmt.Printf("⚠️ unable to summarize %s\n", url)
}

func isDomainDenylisted(url string, denylist []string) bool {
	root := rootDomain(url)
	return slices.Contains(denylist, root)
}
