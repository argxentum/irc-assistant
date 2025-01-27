package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"github.com/gocolly/colly/v2"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const BiasCommandName = "bias"

type BiasCommand struct {
	*commandStub
	retriever retriever.DocumentRetriever
}

func NewBiasCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &BiasCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *BiasCommand) Name() string {
	return BiasCommandName
}

func (c *BiasCommand) Description() string {
	return "Displays source bias and credibility information based on Media Bias Fact Check."
}

func (c *BiasCommand) Triggers() []string {
	return []string{"bias"}
}

func (c *BiasCommand) Usages() []string {
	return []string{"%s <source>"}
}

func (c *BiasCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *BiasCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

var biasRatingRegexp = regexp.MustCompile(`(?m)(?i)bias rating:([^\n]+)`)
var factualReportingRegexp = regexp.MustCompile(`(?m)(?i)factual reporting:([^\n]+)`)
var credibilityRegexp = regexp.MustCompile(`(?m)(?i).*?credibility rating:([^\n]+)`)

func (c *BiasCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	input := strings.Join(tokens[1:], " ")

	if result, ok := repository.GetBiasResultFromAssistantCache(e, input); ok {
		log.Logger().Debugf(e, "found bias result in cache")
		c.SendBiasResult(e, result)
		return
	}

	headers := retriever.RandomHeaderSet()

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), input)

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
			c.Replyf(e, "No bias details found for %s", style.Bold(input))
			return
		}

		logger.Debugf(e, "found detail URL: %s", detailURL)

		err := dc.Visit(detailURL)
		if err != nil {
			logger.Warningf(e, "error visiting detail URL %s, %s", detailURL, err)
			c.Replyf(e, "Unable to determine bias details for %s", style.Bold(input))
			return
		}
	})

	dc.OnHTML("html", func(node *colly.HTMLElement) {
		title := node.DOM.Find("h1").First().Text()
		container := node.DOM.Find("div.entry-content").First()
		detail1 := container.Find("h3 ~ p").First().Text()
		detail2a := container.Find("h3 ~ div.entry-content").First().Text()
		detail2b := container.Find("h3 ~ div.entry-content ~ div.entry-content").First().Text()
		detail2 := detail2a + detail2b

		detail := ""
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

		if len(detail) == 0 {
			logger.Warningf(e, "no bias detail found")
			c.Replyf(e, "No bias details found for %s", style.Bold(input))
			return
		}

		result := models.BiasResult{
			CachedAt: time.Now(),
		}

		result.Title = strings.TrimFunc(title, func(r rune) bool {
			return !unicode.IsLetter(r)
		})
		result.Title = strings.TrimSpace(strings.Replace(title, "Bias and Credibility", "", -1))

		rating := biasRatingRegexp.FindStringSubmatch(detail)
		if len(rating) > 1 {
			content := text.Capitalize(rating[1], true)
			content = strings.TrimFunc(content, func(r rune) bool {
				return !unicode.IsLetter(r)
			})
			result.Rating = text.CapitalizeEveryWord(content, true)
		}

		factual := factualReportingRegexp.FindStringSubmatch(detail)
		if len(factual) > 1 {
			content := text.Capitalize(factual[1], true)
			content = strings.TrimFunc(content, func(r rune) bool {
				return !unicode.IsLetter(r)
			})
			result.Factual = text.CapitalizeEveryWord(content, true)
		}

		credibility := credibilityRegexp.FindStringSubmatch(detail)
		if len(credibility) > 1 {
			content := text.Capitalize(credibility[1], true)
			content = strings.TrimFunc(content, func(r rune) bool {
				return !unicode.IsLetter(r)
			})
			result.Credibility = text.CapitalizeEveryWord(content, true)
		}

		result.DetailURL = strings.TrimSpace(detailURL)

		if err := repository.AddBiasResultToAssistantCache(e, input, result); err != nil {
			logger.Warningf(e, "error adding bias result to cache: %s", err)
		}

		c.SendBiasResult(e, result)
	})

	searchQuery := url.QueryEscape(input)
	err := sc.Visit(fmt.Sprintf("https://mediabiasfactcheck.com/?s=%s", searchQuery))
	if err != nil {
		logger.Warningf(e, "error visiting search URL: %s", err)
		if strings.Contains(strings.ToLower(err.Error()), "too many requests") {
			c.Replyf(e, "Too many requests. Please try again later.")
			return
		}
		c.Replyf(e, "Unable to determine bias details for %s", style.Bold(input))
		return
	}
}

func (c *BiasCommand) SendBiasResult(e *irc.Event, result models.BiasResult) {
	message := ""

	if len(result.Rating) > 0 {
		message += fmt.Sprintf("%s: %s", style.Underline("Bias"), result.Rating)
	}

	if len(result.Factual) > 0 {
		if len(message) > 0 {
			message += ", "
		}
		message += fmt.Sprintf("%s: %s", style.Underline("Factual reporting"), result.Factual)
	}

	if len(result.Credibility) > 0 {
		if len(message) > 0 {
			message += ", "
		}
		message += fmt.Sprintf("%s: %s", style.Underline("Credibility"), result.Credibility)
	}

	if len(message) > 0 {
		message = fmt.Sprintf("%s %s", style.Bold(result.Title), message)
	}

	c.SendMessages(e, e.ReplyTarget(), []string{message, result.DetailURL})
}
