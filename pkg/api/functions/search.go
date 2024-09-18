package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
)

const searchFunctionName = "search"

type searchFunction struct {
	Stub
}

func NewSearchTimeFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, searchFunctionName)
	if err != nil {
		return nil, err
	}

	return &searchFunction{
		Stub: stub,
	}, nil
}

func (f *searchFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *searchFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ search\n")
	/*
		tokens := Tokens(e.Message())
		input := strings.Join(tokens[1:], " ")

		c := colly.NewCollector()
		c.UserAgent = userAgents[rand.Intn(len(userAgents))]
		c.OnHTML("ol.b_results", func(node *colly.HTMLElement) {
			var result *html.Node
			for i, child := range node.DOM.ChildrenFiltered("li.b_algo").Nodes {
				if
			}
		})

		query := url.QueryEscape(input)
		err := c.Visit(fmt.Sprintf("https://www.bing.com/search?q=%s", query))
		if err != nil {
			f.Reply(e, "Unable to search for %s", e.From, text.Bold(input))
			return
		}
	*/
}
