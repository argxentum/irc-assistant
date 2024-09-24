package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/style"
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

func NewSearchFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, searchFunctionName)
	if err != nil {
		return nil, err
	}

	return &searchFunction{
		FunctionStub: stub,
	}, nil
}

func (f *searchFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *searchFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ search\n")
	tokens := Tokens(e.Message())
	input := strings.Join(tokens[1:], " ")
	f.tryBing(e, input)
}

func (f *searchFunction) tryBing(e *core.Event, input string) {
	fmt.Printf("🗒 trying bing for %s\n", input)
	query := url.QueryEscape(input)

	doc, err := getDocument(fmt.Sprintf(bingSearchURL, query), true)
	if err != nil || doc == nil {
		fmt.Printf("⚠️ failed bing search for %s, trying duckduckgo\n", input)
		f.tryDuckDuckGo(e, input)
		return
	}

	container := doc.Find("ol#b_results li.b_algo").First()
	title := strings.TrimSpace(container.Find("h2").First().Text())
	link := strings.TrimSpace(container.Find("h2 a").First().AttrOr("href", ""))
	site := strings.TrimSpace(container.Find("div.tptt").First().Text())

	messages := make([]string, 0)

	if len(title) == 0 || len(link) == 0 {
		fmt.Printf("⚠️ found no bing search results for %s, trying duckduckgo\n", input)
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
		f.irc.SendMessages(e.ReplyTarget(), messages)
	} else {
		fmt.Printf("⚠️ found no bing search results for %s, trying duckduckgo\n", input)
		f.tryDuckDuckGo(e, input)
	}
}

func (f *searchFunction) tryDuckDuckGo(e *core.Event, input string) {
	fmt.Printf("🗒 trying duckduckgo for %s\n", input)
	query := url.QueryEscape(input)

	doc, err := getDocument(fmt.Sprintf(duckDuckGoSearchURL, query), true)
	if err != nil || doc == nil {
		f.Reply(e, "No search results found for %s", style.Bold(input))
		fmt.Printf("⚠️ failed duckduckgo search for %s\n", input)
		return
	}

	title := strings.TrimSpace(doc.Find("div.result__body").First().Find("h2.result__title").First().Text())
	link := strings.TrimSpace(doc.Find("div.result__body").First().Find("h2.result__title").First().Find("a.result__a").First().AttrOr("href", ""))

	if len(title) == 0 || len(link) == 0 {
		fmt.Printf("⚠️ found no duckduckgo search results for %s\n", input)
		f.Reply(e, "No search results found for %s", style.Bold(input))
		return
	}

	f.irc.SendMessage(e.ReplyTarget(), style.Bold(title))
	return
}
