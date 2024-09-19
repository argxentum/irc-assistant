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
	"regexp"
	"strings"
)

const biasFunctionName = "bias"

type biasFunction struct {
	FunctionStub
}

func NewBiasFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, biasFunctionName)
	if err != nil {
		return nil, err
	}

	return &biasFunction{
		FunctionStub: stub,
	}, nil
}

func (f *biasFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

var biasRatingRegexp = regexp.MustCompile(`(?m)(?i)bias rating: ([^\n]+)`)
var factualReportingRegexp = regexp.MustCompile(`(?m)(?i)factual reporting: ([^\n]+)`)
var credibilityRegexp = regexp.MustCompile(`(?m)(?i).*?credibility rating: ([^\n]+)`)

func (f *biasFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ bias\n")
	tokens := Tokens(e.Message())
	input := strings.Join(tokens[1:], " ")
	headers := headerSets[rand.Intn(len(headerSets))]

	sc := colly.NewCollector(
		colly.UserAgent(headers["User-Agent"]),
	)

	dc := sc.Clone()

	sc.OnRequest(func(r *colly.Request) {
		for k, v := range headers {
			r.Headers.Set(k, v)
		}
	})

	detailURL := ""

	sc.OnHTML("html", func(node *colly.HTMLElement) {
		article := node.DOM.Find("article").First()
		detailURL = strings.TrimSpace(article.Find("h3 a").First().AttrOr("href", ""))

		if len(detailURL) == 0 {
			f.Reply(e, "No bias details found for %s", text.Bold(input))
			return
		}

		err := dc.Visit(detailURL)
		if err != nil {
			f.Reply(e, "Unable to determine bias details for %s", text.Bold(input))
			return
		}
	})

	dc.OnHTML("html", func(node *colly.HTMLElement) {
		container := node.DOM.Find("div.entry-content").First()
		summary := strings.TrimSpace(container.Find("ul li strong").First().Text())
		detail := strings.TrimSpace(container.Find("h3 ~ p").First().Text())

		if len(summary) == 0 {
			f.Reply(e, "No bias details found for %s", text.Bold(input))
			return
		}

		f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("MBFC: %s", summary))

		if len(detail) > 0 {
			rating := biasRatingRegexp.FindStringSubmatch(detail)
			if len(rating) > 1 {
				content := strings.ToUpper(rating[1][:1]) + strings.ToLower(rating[1][1:])
				f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("Bias rating: %s", text.Bold(content)))
			}

			factual := factualReportingRegexp.FindStringSubmatch(detail)
			if len(factual) > 1 {
				content := strings.ToUpper(factual[1][:1]) + strings.ToLower(factual[1][1:])
				f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("Factual reporting: %s", text.Bold(content)))
			}

			credibility := credibilityRegexp.FindStringSubmatch(detail)
			if len(credibility) > 1 {
				content := strings.ToUpper(credibility[1][:1]) + strings.ToLower(credibility[1][1:])
				f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("Credibility rating: %s", text.Bold(content)))
			}
		}

		f.irc.SendMessage(e.ReplyTarget(), detailURL)
	})

	searchQuery := url.QueryEscape(input)
	err := sc.Visit(fmt.Sprintf("https://mediabiasfactcheck.com/?s=%s", searchQuery))
	if err != nil {
		f.Reply(e, "Unable to determine bias details for %s", text.Bold(input))
		return
	}
}
