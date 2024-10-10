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
	"net/url"
	"regexp"
	"strings"
)

const searchFunctionName = "search"
const bingSearchURL = "https://www.bing.com/search?q=%s"
const duckDuckGoSearchURL = "https://html.duckduckgo.com/html?q=%s"
const startPageSearchURL = "https://www.startpage.com/sp/search?q=%s"

const duckDuckGoSearchResultURLPattern = `//duckduckgo.com/l/\?uddg=(.*?)&`

type searchFunction struct {
	FunctionStub
	retriever retriever.DocumentRetriever
}

func NewSearchFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, searchFunctionName)
	if err != nil {
		return nil, err
	}

	return &searchFunction{
		FunctionStub: stub,
		retriever:    retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}, nil
}

func (f *searchFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *searchFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	input := strings.Join(tokens[1:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] search %s", e.From, e.ReplyTarget(), input)

	for _, search := range f.searchChain() {
		s, err := search(e, input)
		if err != nil {
			continue
		}

		if s != nil {
			f.SendMessages(e, e.ReplyTarget(), s.messages)
			return
		}
	}
}

var ssf []func(e *irc.Event, input string) (*summary, error)

func (f *searchFunction) searchChain() []func(e *irc.Event, input string) (*summary, error) {
	if ssf == nil {
		ssf = []func(e *irc.Event, input string) (*summary, error){
			f.searchBing,
			f.searchDuckDuckGo,
			f.searchStartPage,
		}
	}

	return ssf
}

func (f *searchFunction) searchBing(e *irc.Event, input string) (*summary, error) {
	logger := log.Logger()
	logger.Debugf(e, "searching bing for %s", input)
	query := url.QueryEscape(input)

	doc, err := f.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(bingSearchURL, query)), retriever.DefaultTimeout)
	if err != nil {
		logger.Debugf(e, "unable to retrieve bing search results for %s: %s", input, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve bing search results for %s", input)
		return nil, fmt.Errorf("bing search results doc nil")
	}

	container := doc.Find("ol#b_results li.b_algo").First()
	title := strings.TrimSpace(container.Find("h2").First().Text())
	link := strings.TrimSpace(container.Find("h2 a").First().AttrOr("href", ""))
	site := strings.TrimSpace(container.Find("div.tptt").First().Text())

	if len(title) == 0 || len(link) == 0 {
		logger.Debugf(e, "empty bing search results for %s, title: %s, link: %s", input, title, link)
		return nil, errors.New("empty bing search results data")
	}

	s := createSummary()

	if len(title) > 0 && len(site) > 0 {
		if strings.Contains(title, site) || strings.Contains(site, title) {
			if len(title) > len(site) {
				s.addMessage(style.Bold(title))
			} else {
				s.addMessage(style.Bold(site))
			}
		} else {
			s.addMessage(fmt.Sprintf("%s • %s", style.Bold(title), site))
		}
	} else if len(site) > 0 {
		s.addMessage(site)
	} else if len(title) > 0 {
		s.addMessage(title)
	} else {
		return nil, summaryTooShortError
	}

	if len(link) == 0 {
		return nil, summaryTooShortError
	}

	s.addMessage(link)

	return s, nil
}

var searchResultURLRegex = regexp.MustCompile(duckDuckGoSearchResultURLPattern)

func (f *searchFunction) searchDuckDuckGo(e *irc.Event, input string) (*summary, error) {
	logger := log.Logger()

	logger.Infof(e, "searching duckduckgo for %s", input)
	query := url.QueryEscape(input)

	doc, err := f.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(duckDuckGoSearchURL, query)), retriever.DefaultTimeout)
	if err != nil {
		logger.Debugf(e, "unable to retrieve duckduckgo search results for %s: %s", input, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve duckduckgo search results for %s", input)
		return nil, fmt.Errorf("duckduckgo search results doc nil")
	}

	title := strings.TrimSpace(doc.Find("div.result__body h2.result__title").First().Text())
	linkRaw := strings.TrimSpace(doc.Find("div.result__body h2.result__title a.result__a").First().AttrOr("href", ""))

	match := searchResultURLRegex.FindStringSubmatch(linkRaw)
	if len(match) < 2 {
		logger.Debugf(e, "unable to parse duckduckgo search result link for %s", input)
		return nil, summaryTooShortError
	}

	link, err := url.QueryUnescape(match[1])
	if err != nil {
		logger.Debugf(e, "unable to unescape duckduckgo search result link for %s: %s", input, err)
		return nil, err
	}

	if len(title) == 0 || len(link) == 0 {
		logger.Debugf(e, "empty duckduckgo search results for %s", input)
		return nil, summaryTooShortError
	}

	return createSummary(style.Bold(title), link), nil
}

func (f *searchFunction) searchStartPage(e *irc.Event, input string) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "searching startpage for %s", input)
	query := url.QueryEscape(input)

	doc, err := f.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(startPageSearchURL, query)), retriever.DefaultTimeout)
	if err != nil {
		logger.Debugf(e, "unable to retrieve startpage search results for %s: %s", input, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve startpage search results for %s", input)
		return nil, fmt.Errorf("startpage search results doc nil")
	}

	link := doc.Find("section#main a.result-link").First().AttrOr("href", "")
	title := strings.TrimSpace(doc.Find("section#main h2").First().Text())

	if len(title) == 0 || len(link) == 0 {
		logger.Debugf(e, "empty startpage search results for %s", input)
		return nil, summaryTooShortError
	}

	return createSummary(style.Bold(title), link), nil
}
