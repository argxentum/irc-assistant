package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strings"
)

const pollsFunctionName = "polls"
const pollsURL = "https://www.270towin.com/2024-presidential-election-polls/national"

type pollsFunction struct {
	FunctionStub
}

func NewPollsFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, pollsFunctionName)
	if err != nil {
		return nil, err
	}

	return &pollsFunction{
		FunctionStub: stub,
	}, nil
}

func (f *pollsFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 0)
}

func (f *pollsFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ polls\n")

	doc, err := getDocument(pollsURL, true)
	if err != nil || doc == nil {
		f.Reply(e, "Unable to retrieve polling data")
		return
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())
	subtitle := strings.TrimSpace(doc.Find("h3").First().Text())

	candidates := make([]string, 0)
	doc.Find("table#polls").First().Find("th").Each(func(i int, s *goquery.Selection) {
		if _, ok := s.Attr("candidate_id"); ok {
			candidates = append(candidates, strings.TrimSpace(s.Text()))
		}
	})

	averages := make([]string, 0)
	average := doc.Find("table#polls").First().Find("td.poll_avg").First()
	for range candidates {
		average = average.Next()
		averages = append(averages, strings.TrimSpace(average.Text()))
	}

	if len(candidates) != len(averages) || len(candidates) == 0 {
		f.Reply(e, "Unable to parse polling data")
		return
	}

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s: %s", title, text.Bold(subtitle)))
	for i, c := range candidates {
		messages = append(messages, fmt.Sprintf("%s: %s", text.Bold(text.Underline(c)), text.Bold(averages[i])))
	}
	messages = append(messages, pollsURL)

	f.irc.SendMessages(e.ReplyTarget(), messages)
}
