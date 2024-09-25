package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"github.com/gocolly/colly/v2"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

const biasFunctionName = "bias"

type biasFunction struct {
	FunctionStub
}

func NewBiasFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, biasFunctionName)
	if err != nil {
		return nil, err
	}

	return &biasFunction{
		FunctionStub: stub,
	}, nil
}

func (f *biasFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

var biasRatingRegexp = regexp.MustCompile(`(?m)(?i)bias rating:([^\n]+)`)
var factualReportingRegexp = regexp.MustCompile(`(?m)(?i)factual reporting:([^\n]+)`)
var credibilityRegexp = regexp.MustCompile(`(?m)(?i).*?credibility rating:([^\n]+)`)

func (f *biasFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	input := strings.Join(tokens[1:], " ")
	headers := headerSets[rand.Intn(len(headerSets))]

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] bias %s", e.From, e.ReplyTarget(), input)

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
			logger.Debug(e, "no detail URL found")
			f.Replyf(e, "No bias details found for %s", style.Bold(input))
			return
		}

		logger.Debugf(e, "found detail URL: %s", detailURL)

		err := dc.Visit(detailURL)
		if err != nil {
			logger.Warningf(e, "error visiting detail URL %s, %s", detailURL, err)
			f.Replyf(e, "Unable to determine bias details for %s", style.Bold(input))
			return
		}
	})

	dc.OnHTML("html", func(node *colly.HTMLElement) {
		container := node.DOM.Find("div.entry-content").First()
		summary := container.Find("ul li strong").First().Text()
		detail1 := container.Find("h3 ~ p").First().Text()
		detail2a := container.Find("h3 ~ div.entry-content").First().Text()
		detail2b := container.Find("h3 ~ div.entry-content ~ div.entry-content").First().Text()
		detail2 := detail2a + detail2b

		detail := ""
		summary = strings.TrimSpace(summary)
		detail1Lower := strings.ToLower(detail1)
		detail2Lower := strings.ToLower(detail2)
		if strings.Contains(detail1Lower, "bias rating") && strings.Contains(detail1Lower, "credibility rating") {
			detail = strings.TrimSpace(detail1)
		} else if strings.Contains(detail2Lower, "bias rating") && strings.Contains(detail2Lower, "credibility rating") {
			detail = strings.TrimSpace(detail2)
		}

		detail = strings.TrimFunc(detail, func(r rune) bool {
			return !unicode.IsGraphic(r)
		})

		if len(summary) == 0 {
			logger.Warningf(e, "no bias summary found")
			f.Replyf(e, "No bias details found for %s", style.Bold(input))
			return
		}

		messages := make([]string, 0)
		messages = append(messages, fmt.Sprintf("MBFC: %s", summary))

		if len(detail) > 0 {
			t := createDefaultTable()

			rating := biasRatingRegexp.FindStringSubmatch(detail)
			if len(rating) > 1 {
				content := strings.ToUpper(rating[1][:1]) + strings.ToLower(rating[1][1:])
				t.AppendRow([]any{"Bias rating", style.Bold(content)})
			}

			factual := factualReportingRegexp.FindStringSubmatch(detail)
			if len(factual) > 1 {
				content := strings.ToUpper(factual[1][:1]) + strings.ToLower(factual[1][1:])
				t.AppendRow([]any{"Factual reporting", style.Bold(content)})
			}

			credibility := credibilityRegexp.FindStringSubmatch(detail)
			if len(credibility) > 1 {
				content := strings.ToUpper(credibility[1][:1]) + strings.ToLower(credibility[1][1:])
				t.AppendRow([]any{"Credibility rating", style.Bold(content)})
			}

			messages = append(messages, strings.Split(t.Render(), "\n")...)
		}

		messages = append(messages, detailURL)

		f.SendMessages(e, e.ReplyTarget(), messages)
	})

	searchQuery := url.QueryEscape(input)
	err := sc.Visit(fmt.Sprintf("https://mediabiasfactcheck.com/?s=%s", searchQuery))
	if err != nil {
		logger.Warningf(e, "error visiting search URL: %s", err)
		f.Replyf(e, "Unable to determine bias details for %s", style.Bold(input))
		return
	}
}
