package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
	"github.com/gocolly/colly/v2"
	"math/rand"
	"net/url"
	"strings"
)

const searchFunctionName = "search"

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
	headers := headerSets[rand.Intn(len(headerSets))]

	c := colly.NewCollector(
		colly.UserAgent(headers["User-Agent"]),
	)

	c.OnRequest(func(r *colly.Request) {
		for k, v := range headers {
			r.Headers.Set(k, v)
		}
	})

	c.OnHTML("html", func(node *colly.HTMLElement) {
		container := node.DOM.Find("ol#b_results li.b_algo").First()
		title := strings.TrimSpace(container.Find("h2").First().Text())
		link := strings.TrimSpace(container.Find("h2 a").First().AttrOr("href", ""))
		site := strings.TrimSpace(container.Find("div.tptt").First().Text())

		messages := make([]string, 0)

		if len(link) == 0 {
			f.Reply(e, "No search results found for %s", text.Bold(input))
			fmt.Printf("❌ %s\n", c.UserAgent)
			return
		}

		if len(title) > 0 && len(site) > 0 {
			if strings.Contains(title, site) || strings.Contains(site, title) {
				if len(title) > len(site) {
					messages = append(messages, text.Bold(title))
				} else {
					messages = append(messages, text.Bold(site))
				}
			} else {
				messages = append(messages, fmt.Sprintf("%s: %s", site, text.Bold(title)))
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
			f.Reply(e, "No search results found for %s", text.Bold(input))
		}
	})

	query := url.QueryEscape(input)
	err := c.Visit(fmt.Sprintf("https://www.bing.com/search?q=%s", query))
	if err != nil {
		f.Reply(e, "Unable to search for %s", e.From, text.Bold(input))
		return
	}
}
