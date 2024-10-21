package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"errors"
	"regexp"
	"slices"
	"strings"
)

const summaryCommandName = "summary"

const minimumTitleLength = 16
const minimumPreferredTitleLength = 64
const maximumDescriptionLength = 300

type summary struct {
	messages []string
}

func createSummary(message ...string) *summary {
	m := make([]string, 0)
	if len(message) > 0 {
		m = append(m, message...)
	}

	return &summary{messages: m}
}

func (s *summary) addMessage(message string) {
	s.messages = append(s.messages, message)
}

type summaryCommand struct {
	*commandStub
	bodyRetriever retriever.BodyRetriever
	docRetriever  retriever.DocumentRetriever
}

func NewSummaryCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &summaryCommand{
		commandStub:   defaultCommandStub(ctx, cfg, irc),
		bodyRetriever: retriever.NewBodyRetriever(),
		docRetriever:  retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *summaryCommand) Name() string {
	return summaryCommandName
}

func (c *summaryCommand) Description() string {
	return "Displays a summary of the content at the given URL."
}

func (c *summaryCommand) Triggers() []string {
	return []string{}
}

func (c *summaryCommand) Usages() []string {
	return []string{"<url>"}
}

func (c *summaryCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *summaryCommand) CanExecute(e *irc.Event) bool {
	if !c.isCommandEventValid(c, e, 0) {
		return false
	}

	message := e.Message()
	return strings.Contains(message, "https://") || strings.Contains(message, "http://")
}

func (c *summaryCommand) Execute(e *irc.Event) {
	logger := log.Logger()

	url := parseURLFromMessage(e.Message())
	if len(url) == 0 {
		logger.Debugf(e, "no URL found in message")
		return
	}

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), url)

	if c.isRootDomainIn(url, c.cfg.Ignore.Domains) {
		logger.Debugf(e, "root domain denied %s", url)
		return
	}

	if c.isDomainIn(url, domainDenylist) {
		logger.Debugf(e, "domain denied %s", url)
		return
	}

	if c.requiresDomainSummary(url) {
		logger.Debugf(e, "performing domain summarization for %s", url)

		ds, err := c.domainSummary(e, url)
		if err != nil {
			logger.Debugf(e, "domain specific summarization failed for %s: %s", url, err)
		} else if ds != nil {
			logger.Debugf(e, "performed domain specific handling: %s", url)
			c.SendMessages(e, e.ReplyTarget(), ds.messages)
		} else {
			logger.Debugf(e, "domain specific summarization failed for %s", url)
		}
		return
	}

	cs, err := c.contentSummary(e, url)
	if err != nil {
		if errors.Is(err, retriever.DisallowedContentTypeError) {
			logger.Debugf(e, "disallowed content type for %s", url)
			return
		}
	}

	if cs != nil {
		logger.Debugf(e, "performing content summarization for %s", url)

		s, err := cs(e, url)
		if err != nil {
			logger.Debugf(e, "content specific summarization failed for %s: %s", url, err)
		}
		if s != nil {
			c.SendMessages(e, e.ReplyTarget(), s.messages)
			return
		}
	}

	s, err := c.requestSummary(e, url)
	if err != nil {
		logger.Debugf(e, "unable to summarize %s: %s", url, err)
	}

	if s == nil {
		logger.Debugf(e, "unable to summarize %s", url)
	} else {
		c.SendMessages(e, e.ReplyTarget(), s.messages)
	}
}

func parseURLFromMessage(message string) string {
	urlRegex := regexp.MustCompile(`(?i)(https?://\S+)\b`)
	urlMatches := urlRegex.FindStringSubmatch(message)
	if len(urlMatches) > 0 {
		return urlMatches[0]
	}
	return ""
}

var rejectedTitlePrefixes = []string{
	"just a moment",
	"sorry, you have been blocked",
	"access to this page has been denied",
	"please verify you are a human",
	"you are being redirected",
	"whoa there, pardner",
	"page not found",
}

var domainDenylist = []string{
	"i.redd.it",
}

func (c *summaryCommand) isRootDomainIn(url string, domains []string) bool {
	root := retriever.RootDomain(url)
	return slices.Contains(domains, root)
}

func (c *summaryCommand) isDomainIn(url string, domains []string) bool {
	domain := retriever.Domain(url)
	return slices.Contains(domains, domain)
}

func isRejectedTitle(title string) bool {
	for _, prefix := range rejectedTitlePrefixes {
		if strings.HasPrefix(strings.ToLower(title), prefix) {
			return true
		}
	}
	return false
}
