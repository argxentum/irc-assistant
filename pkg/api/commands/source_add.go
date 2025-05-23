package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"regexp"
	"strings"
	"unicode"
)

const SourceAddCommandName = "add_source"

type SourceAddCommand struct {
	*commandStub
}

func NewSourceAddCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &SourceAddCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *SourceAddCommand) Name() string {
	return SourceAddCommandName
}

func (c *SourceAddCommand) Description() string {
	return "Add source bias and credibility information."
}

func (c *SourceAddCommand) Triggers() []string {
	return []string{"sourceadd"}
}

func (c *SourceAddCommand) Usages() []string {
	return []string{"%s <mbfc-url> <source-domain> [<keyword>...]"}
}

func (c *SourceAddCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *SourceAddCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 2)
}

var biasRatingRegexp = regexp.MustCompile(`(?m)(?i)bias rating:([^\n]+)`)
var factualityReportingRegexp = regexp.MustCompile(`(?m)(?i)factual reporting:([^\n]+)`)
var credibilityRegexp = regexp.MustCompile(`(?m)(?i).*?credibility rating:([^\n]+)`)

func (c *SourceAddCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())

	url := tokens[1]
	domain := strings.ToLower(tokens[2])
	keywords := make([]string, 0)
	if len(tokens) > 3 {
		keywords = tokens[3:]
	}
	for i, keyword := range keywords {
		keywords[i] = strings.ToLower(keyword)
	}

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), url)

	if s, err := repository.FindSource(domain); err == nil && s != nil {
		c.Replyf(e, "Source details already exist for %s", style.Bold(domain))
		return
	}

	dr := retriever.NewDocumentRetriever(retriever.NewBodyRetriever())
	params := retriever.DefaultParams(url)
	params.Timeout = 2000
	doc, err := dr.RetrieveDocumentSelection(e, params, "html")
	if err != nil || doc == nil {
		if err != nil {
			logger.Warningf(e, "unable to parse url (%s): %s", url, err)
		} else {
			logger.Warningf(e, "unable to parse url (%s)", url)
		}
		c.Replyf(e, "Unable to retrieve source details from %s", style.Bold(url))
		return
	}

	title := doc.Find("h1").First().Text()
	container := doc.Find("div.entry-content").First()
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
		logger.Warningf(e, "no source details found")
		c.Replyf(e, "No source details found for %s", style.Bold(url))
		return
	}

	source := models.NewEmptySource()

	source.Title = strings.TrimFunc(title, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	source.Title = strings.TrimSpace(strings.Replace(title, "Bias and Credibility", "", -1))
	source.Title = strings.TrimSpace(strings.Replace(source.Title, "–", "", -1))

	bias := biasRatingRegexp.FindStringSubmatch(detail)
	if len(bias) > 1 {
		content := strings.ToLower(bias[1])
		content = strings.TrimFunc(content, func(r rune) bool {
			return !unicode.IsLetter(r)
		})
		source.Bias = content
	}

	factuality := factualityReportingRegexp.FindStringSubmatch(detail)
	if len(factuality) > 1 {
		content := strings.ToLower(factuality[1])
		content = strings.TrimFunc(content, func(r rune) bool {
			return !unicode.IsLetter(r)
		})
		source.Factuality = content
	}

	credibility := credibilityRegexp.FindStringSubmatch(detail)
	if len(credibility) > 1 {
		content := strings.ToLower(credibility[1])
		content = strings.TrimFunc(content, func(r rune) bool {
			return !unicode.IsLetter(r)
		})
		source.Credibility = content
	}

	source.Reviews = append(source.Reviews, strings.TrimSpace(url))
	source.URLs = append(source.URLs, domain)

	for _, keyword := range keywords {
		source.Keywords = append(source.Keywords, keyword)
	}

	if err := repository.AddSource(source); err != nil {
		logger.Warningf(e, "error adding source details: %s", err)
	}

	SendSource(c.commandStub, e, source)
}
