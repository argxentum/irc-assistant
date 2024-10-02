package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const summaryFunctionName = "summary"

type summaryFunction struct {
	FunctionStub
	retriever retriever.DocumentRetriever
}

func NewSummaryFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, summaryFunctionName)
	if err != nil {
		return nil, err
	}

	return &summaryFunction{
		FunctionStub: stub,
		retriever:    retriever.NewDocumentRetriever(),
	}, nil
}

func (f *summaryFunction) MayExecute(e *irc.Event) bool {
	if !f.isValid(e, 0) {
		return false
	}

	message := e.Message()
	return strings.Contains(message, "https://") || strings.Contains(message, "http://")
}

func (f *summaryFunction) Execute(e *irc.Event) {
	logger := log.Logger()

	url := parseURLFromMessage(e.Message())
	if len(url) == 0 {
		logger.Debugf(e, "no URL found in message")
		return
	}

	logger.Infof(e, "⚡ [%s/%s] summary %s", e.From, e.ReplyTarget(), url)
	f.tryDirect(e, url, false)
}

func parseURLFromMessage(message string) string {
	urlRegex := regexp.MustCompile(`(?i)(https?://\S+)\b`)
	urlMatches := urlRegex.FindStringSubmatch(message)
	if len(urlMatches) > 0 {
		return urlMatches[0]
	}
	return ""
}

const minimumTitleLength = 16
const minimumPreferredTitleLength = 64
const maximumPreferredTitleLength = 128
const maximumDescriptionLength = 256

var rejectedTitlePrefixes = []string{
	"just a moment",
	"sorry, you have been blocked",
	"access to this page has been denied",
	"please verify you are a human",
	"you are being redirected",
}

var domainDenylist = []string{
	"imgur.com",
}

var descriptionDomainDenylist = []string{
	"youtube.com",
	"youtu.be",
}

func (f *summaryFunction) tryDirect(e *irc.Event, url string, impersonated bool) {
	logger := log.Logger()
	logger.Infof(e, "trying direct (impersonated: %t) for %s", impersonated, url)

	if f.isDomainDenylisted(url, domainDenylist) {
		logger.Debugf(e, "domain denylisted %s", url)
		return
	}

	params := retriever.DefaultParams(url)
	params.Impersonate = impersonated

	doc, err := f.retriever.RetrieveDocument(e, params, retriever.DefaultTimeout)
	if err != nil || doc == nil {
		if err != nil {
			if errors.Is(err, retriever.DisallowedContentTypeError) {
				logger.Debugf(e, "disallowed content type for %s", url)
				return
			}
			logger.Debugf(e, "unable to retrieve %s (impersonated: %t): %s", url, impersonated, err)
		} else {
			logger.Debugf(e, "unable to retrieve %s (impersonated: %t)", url, impersonated)
		}

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

	if len(description) > maximumDescriptionLength {
		description = description[:maximumDescriptionLength] + "..."
	}

	if len(titleAttr) > 0 {
		title = titleMeta
	} else if len(h1) > 0 {
		title = h1
	}

	if isRejectedTitle(title) {
		logger.Debugf(e, "rejected title: %s", title)
		f.tryNuggetize(e, url)
		return
	}

	if len(title)+len(description) < minimumTitleLength {
		logger.Debugf(e, "title and description too short, title: %s, description: %s", title, description)
		f.tryNuggetize(e, url)
		return
	}

	includeDescription := true
	if f.isDomainDenylisted(url, descriptionDomainDenylist) {
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

	doc, err := f.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf("https://nug.zip/%s", url)), retriever.DefaultTimeout)
	if err != nil || doc == nil {
		if err != nil {
			logger.Debugf(e, "unable to retrieve nuggetize summary for %s: %s", url, err)
		} else {
			logger.Debugf(e, "unable to retrieve nuggetize summary for %s", url)
		}
		f.tryBing(e, url)
		return
	}

	title := strings.TrimSpace(doc.Find("span.title").First().Text())

	if isRejectedTitle(title) {
		logger.Debugf(e, "rejected title: %s", title)
		f.tryBing(e, url)
		return
	}

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

	if f.isDomainDenylisted(url, bingDomainDenylist) {
		logger.Debugf(e, "bing domain denylisted %s", url)
		f.tryDuckDuckGo(e, url)
		return
	}

	doc, err := f.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(bingSearchURL, url)), retriever.DefaultTimeout)
	if err != nil || doc == nil {
		if err != nil {
			logger.Debugf(e, "unable to retrieve bing search results for %s: %s", url, err)
		} else {
			logger.Debugf(e, "unable to retrieve bing search results for %s", url)
		}
		f.tryDuckDuckGo(e, url)
		return
	}

	title := strings.TrimSpace(doc.Find("ol#b_results").First().Find("h2").First().Text())

	if isRejectedTitle(title) {
		logger.Debugf(e, "rejected title: %s", title)
		f.tryDuckDuckGo(e, url)
		return
	}

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "bing title contains url': %s", title)
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

	if f.isDomainDenylisted(url, duckDuckGoDomainDenylist) {
		logger.Debugf(e, "duckduckgo domain denylisted %s", url)
		return
	}

	doc, err := f.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(duckDuckGoSearchURL, url)), retriever.DefaultTimeout)
	if err != nil || doc == nil {
		if err != nil {
			logger.Debugf(e, "unable to retrieve duckduckgo search results for %s: %s", url, err)
		} else {
			logger.Debugf(e, "unable to retrieve duckduckgo search results for %s", url)
		}
		return
	}

	title := strings.TrimSpace(doc.Find("div.result__body").First().Find("h2.result__title").First().Text())

	if strings.Contains(strings.ToLower(title), url[:min(len(url), 24)]) {
		logger.Debugf(e, "duckduckgo title contains url: %s", title)
		return
	}

	if len(title) > 0 {
		f.SendMessage(e, e.ReplyTarget(), style.Bold(title))
		return
	}

	logger.Debugf(e, "unable to summarize %s", url)
}

func (f *summaryFunction) isDomainDenylisted(url string, denylist []string) bool {
	root := retriever.RootDomain(url)
	return slices.Contains(denylist, root)
}

func isRejectedTitle(title string) bool {
	for _, prefix := range rejectedTitlePrefixes {
		if strings.HasPrefix(strings.ToLower(title), prefix) {
			return true
		}
	}
	return false
}
