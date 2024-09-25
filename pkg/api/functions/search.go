package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"net/url"
	"strings"
)

const searchFunctionName = "search"
const bingSearchURL = "https://www.bing.com/search?q=%s"
const duckDuckGoSearchURL = "https://html.duckduckgo.com/html?q=%s"

type searchFunction struct {
	FunctionStub
}

func NewSearchFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, searchFunctionName)
	if err != nil {
		return nil, err
	}

	return &searchFunction{
		FunctionStub: stub,
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

	f.tryBing(e, input)
}

func (f *searchFunction) tryBing(e *irc.Event, input string) {
	logger := log.Logger()
	logger.Debugf(e, "trying bing for %s", input)
	query := url.QueryEscape(input)

	doc, err := getDocument(fmt.Sprintf(bingSearchURL, query), true)
	if err != nil || doc == nil {
		logger.Warningf(e, "unable to retrieve bing search results for %s: %s", input, err)
		f.tryDuckDuckGo(e, input)
		return
	}

	container := doc.Find("ol#b_results li.b_algo").First()
	title := strings.TrimSpace(container.Find("h2").First().Text())
	link := strings.TrimSpace(container.Find("h2 a").First().AttrOr("href", ""))
	site := strings.TrimSpace(container.Find("div.tptt").First().Text())

	messages := make([]string, 0)

	if len(title) == 0 || len(link) == 0 {
		logger.Warningf(e, "unable to parse bing search results for %s, title: %s, link: %s", input, title, link)
		f.tryDuckDuckGo(e, input)
		return
	}

	if len(title) > 0 && len(site) > 0 {
		if strings.Contains(title, site) || strings.Contains(site, title) {
			if len(title) > len(site) {
				messages = append(messages, style.Bold(title))
			} else {
				messages = append(messages, style.Bold(site))
			}
		} else {
			messages = append(messages, fmt.Sprintf("%s: %s", site, style.Bold(title)))
		}
	} else if len(site) > 0 {
		messages = append(messages, site)
	} else if len(title) > 0 {
		messages = append(messages, title)
	}

	if len(link) > 0 {
		messages = append(messages, link)
	}

	if len(messages) > 0 {
		f.SendMessages(e, e.ReplyTarget(), messages)
	} else {
		logger.Warningf(e, "no bing search results for %s", input)
		f.tryDuckDuckGo(e, input)
	}
}

func (f *searchFunction) tryDuckDuckGo(e *irc.Event, input string) {
	logger := log.Logger()

	logger.Infof(e, "trying duckduckgo for %s", input)
	query := url.QueryEscape(input)

	doc, err := getDocument(fmt.Sprintf(duckDuckGoSearchURL, query), true)
	if err != nil || doc == nil {
		logger.Warningf(e, "unable to retrieve duckduckgo search results for %s: %s", input, err)
		f.Replyf(e, "No search results found for %s", style.Bold(input))
		return
	}

	title := strings.TrimSpace(doc.Find("div.result__body").First().Find("h2.result__title").First().Text())
	link := strings.TrimSpace(doc.Find("div.result__body").First().Find("h2.result__title").First().Find("a.result__a").First().AttrOr("href", ""))

	if len(title) == 0 || len(link) == 0 {
		logger.Warningf(e, "unable to parse duckduckgo search results for %s", input)
		f.Replyf(e, "No search results found for %s", style.Bold(input))
		return
	}

	f.SendMessage(e, e.ReplyTarget(), style.Bold(title))
	return
}
