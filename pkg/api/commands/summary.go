package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"errors"
	"fmt"
	"html"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/bobesa/go-domain-util/domainutil"
)

const SummaryCommandName = "summary"

const minimumTitleLength = 16
const maximumTitleLength = 256
const minimumPreferredTitleLength = 64
const standardMaximumDescriptionLength = 300
const extendedMaximumDescriptionLength = 350
const startPauseTimeoutSeconds = 5
const maxPauseTimeoutSeconds = 600
const pauseSummaryMultiplier = 1.025
const pauseDisinfoMultiplier = 2.5
const pauseShowWarningAfter = 5
const disinfoKickThreshold = 3
const disinfoTempMuteDuration = "5m"

type summary struct {
	messages []string
}

type UserPause struct {
	channel      string
	nick         string
	summaryCount int
	disinfoCount int
	timeoutAt    time.Time
	ignoreCount  int
}

func createSummary(messages ...string) *summary {
	m := make([]string, 0)
	for i := 0; i < len(messages); i++ {
		messages[i] = html.UnescapeString(messages[i])
		if len(messages[i]) > 0 {
			m = append(m, messages[i])
		}
	}

	return &summary{messages: m}
}

func (s *summary) addMessage(message string) {
	s.messages = append(s.messages, html.UnescapeString(message))
}

func (s *summary) addMessages(messages ...string) {
	m := make([]string, 0)
	for i := 0; i < len(messages); i++ {
		messages[i] = html.UnescapeString(messages[i])
		if len(messages[i]) > 0 {
			m = append(m, messages[i])
		}
	}
	if len(m) > 0 {
		s.messages = append(s.messages, m...)
	}
}

type SummaryCommand struct {
	*commandStub
	bodyRetriever retriever.BodyRetriever
	docRetriever  retriever.DocumentRetriever
	userPauses    map[string]*UserPause
}

func NewSummaryCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &SummaryCommand{
		commandStub:   defaultCommandStub(ctx, cfg, irc),
		bodyRetriever: retriever.NewBodyRetriever(),
		docRetriever:  retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
		userPauses:    make(map[string]*UserPause),
	}
}

func (c *SummaryCommand) Name() string {
	return SummaryCommandName
}

func (c *SummaryCommand) Description() string {
	return "Displays a summary of the content at the given URL."
}

func (c *SummaryCommand) Triggers() []string {
	return []string{}
}

func (c *SummaryCommand) Usages() []string {
	return []string{"<url>"}
}

func (c *SummaryCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *SummaryCommand) CanExecute(e *irc.Event) bool {
	if !c.isCommandEventValid(c, e, 0) {
		return false
	}

	message := e.Message()
	return strings.Contains(message, "https://") || strings.Contains(message, "http://")
}

func (c *SummaryCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	fs := firestore.Get()
	p := c.userPauses[e.From+"@"+e.ReplyTarget()]

	channel, err := fs.Channel(e.ReplyTarget())
	if err != nil {
		logger.Errorf(e, "error retrieving channel, %s", err)
		return
	}

	url := parseURLFromMessage(e.Message())
	if len(url) == 0 {
		logger.Debugf(e, "no URL found in message")
		return
	}

	// some urls are used to avoid specific domains, e.g., xcancel.com to avoid x.com
	if actualURL, ok := c.actualURL(url); ok {
		logger.Debugf(e, "actualURL %s to %s", url, actualURL)
		url = actualURL
	}

	originalURL := url

	// we need to translate some domains to get the actual content, e.g., x.com to fixupx.com
	if translatedURL, ok := c.translatedURL(url); ok {
		logger.Debugf(e, "translatedURL %s to %s", url, translatedURL)
		url = translatedURL
	}

	source, err := repository.FindSource(url)
	if err != nil {
		logger.Errorf(nil, "error finding source, %s", err)
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

	dis := false
	if channel != nil && fs.IsDisinformationSource(channel.Name, originalURL) {
		dis = true
		logger.Debugf(e, "URL is possible disinformation: %s", url)
	}

	if c.requiresDomainSummary(url) {
		logger.Debugf(e, "performing domain summarization for %s", url)

		ds, err := c.domainSummary(e, url)
		if err != nil {
			logger.Debugf(e, "domain specific summarization failed for %s: %s", url, err)
		} else if ds != nil {
			logger.Debugf(e, "performed domain specific handling: %s", url)

			if dis {
				ds.messages = append(ds.messages, "⚠️ Possible disinformation, use caution.")
			}
			c.completeSummary(e, source, url, e.ReplyTarget(), ds.messages, dis, p)
		} else {
			logger.Debugf(e, "domain specific summarization failed for %s", url)
		}
		return
	}

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(url))
	if err != nil {
		logger.Debugf(e, "error retrieving document for %s: %v", url, err)
		return
	}

	if !retriever.IsContentTypeAllowed(doc.Body.Response.Header.Get("Content-Type")) {
		logger.Debugf(e, "direct prefetch, disallowed content type for %s", url)
		return
	}

	canonicalLink, _ := doc.Root.Find("link[rel='canonical']").First().Attr("href")
	if isValidCanonicalLink(url, canonicalLink) {
		url = canonicalLink
		doc, err = c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(url))
		if err != nil {
			logger.Debugf(e, "error retrieving canonical document for %s: %v", url, err)
			return
		}
	}

	if !e.IsPrivateMessage() && p != nil {
		if p.timeoutAt.After(time.Now()) {
			logger.Debugf(e, "ignoring paused summary request from %s in %s", e.From, e.ReplyTarget())
			if dis {
				c.SendMessage(e, e.ReplyTarget(), "⚠️ Possible disinformation, use caution.")
			}

			p.ignoreCount++
			p.summaryCount++
			if dis {
				p.disinfoCount++
			}
			updatePause(e, p)
			if dis && p.disinfoCount >= disinfoKickThreshold {
				c.ExecuteSynthesizedEvent(e, TempMuteCommandName, fmt.Sprintf("%s %s disinformation threshold reached", disinfoTempMuteDuration, e.From))
				return
			}
			if p.ignoreCount > pauseShowWarningAfter {
				c.Replyf(e, "%s Slow down, please. I've paused summarizing your links for %s.", "\U0001F975", elapse.FutureTimeDescriptionConcise(p.timeoutAt))
			}
			c.userPauses[e.From+"@"+e.ReplyTarget()] = p
			return
		} else {
			logger.Debugf(e, "pause expired for %s in %s", e.From, e.ReplyTarget())
			p.timeoutAt = time.Time{}
			p.summaryCount = 0
			p.disinfoCount = 0
			p.ignoreCount = 0
		}
	}

	contentSummarizer, err := c.contentSummary(e, doc)
	if err != nil {
		if errors.Is(err, retriever.DisallowedContentTypeError) {
			logger.Debugf(e, "disallowed content type for %s", url)
			return
		}
	}

	if contentSummarizer != nil {
		logger.Debugf(e, "performing content summarization for %s", url)

		s, err := contentSummarizer(e, doc)
		if err != nil {
			logger.Debugf(e, "content specific summarization failed for %s: %s", url, err)
		}
		if s != nil {
			messages := s.messages

			if source != nil {
				messages = append(messages, repository.ShortSourceSummary(source))
			}

			c.completeSummary(e, source, url, e.ReplyTarget(), messages, dis, p)
			return
		}
	}

	s, err := c.summarize(e, doc)
	if err != nil {
		logger.Debugf(e, "unable to summarize %s: %s", url, err)
	}

	if s == nil {
		logger.Debugf(e, "unable to summarize %s", url)
	} else {
		if dis {
			s.messages = append(s.messages, "⚠️ Possible disinformation, use caution.")
		}
		c.completeSummary(e, source, url, e.ReplyTarget(), s.messages, dis, p)
	}
}

func isValidCanonicalLink(original, canonical string) bool {
	return len(canonical) > 0 && canonical != original && strings.HasPrefix(strings.ToLower(canonical), "https://")
}

func (c *SummaryCommand) InitializeUserPause(channel, nick string, duration time.Duration) *UserPause {
	logger := log.Logger()

	p := c.userPauses[nick+"@"+channel]
	if p == nil || p.timeoutAt.Before(time.Now()) {
		p = &UserPause{
			channel:     channel,
			nick:        nick,
			timeoutAt:   time.Now().Add(duration),
			ignoreCount: 0,
		}
		c.userPauses[nick+"@"+channel] = p
		logger.Debugf(nil, "join, pausing %s in %s until %s", nick, channel, elapse.TimeDescription(p.timeoutAt))
	} else {
		logger.Debugf(nil, "join, pause already in effect for %s in %s until %s", nick, channel, elapse.TimeDescription(p.timeoutAt))
	}

	return p
}

var escapedHtmlEntityRegex = regexp.MustCompile(`&[a-zA-Z0-9]+;`)

func (c *SummaryCommand) completeSummary(e *irc.Event, source *models.Source, url, target string, messages []string, dis bool, p *UserPause) {
	if !e.IsPrivateMessage() {
		if p == nil {
			p = &UserPause{
				channel: target,
				nick:    e.From,
			}
		}
		p.summaryCount++
		if dis {
			p.disinfoCount++
		}
		updatePause(e, p)
		if dis && p.disinfoCount >= disinfoKickThreshold {
			c.ExecuteSynthesizedEvent(e, TempMuteCommandName, fmt.Sprintf("%s %s disinformation threshold reached", disinfoTempMuteDuration, e.From))
			return
		}
		c.userPauses[e.From+"@"+target] = p
	}

	unescapedMessages := make([]string, 0)
	for _, message := range messages {
		if escapedHtmlEntityRegex.MatchString(message) {
			unescapedMessages = append(unescapedMessages, html.UnescapeString(message))
		} else {
			unescapedMessages = append(unescapedMessages, message)
		}
	}

	if source != nil {
		sourceSummary := repository.ShortSourceSummary(source)
		if source.Paywall && c.isRootDomainIn(url, source.URLs) {
			id, err := repository.GetArchiveShortcutID(url)
			if err == nil && len(id) > 0 {
				sourceSummary += " | " + "\U0001F513 " + fmt.Sprintf(shortcutURLPattern, c.cfg.Web.ExternalRootURL) + id
			}
		}
		unescapedMessages = append(unescapedMessages, sourceSummary)
	}

	c.SendMessages(e, target, unescapedMessages)
}

func updatePause(e *irc.Event, p *UserPause) {
	logger := log.Logger()
	sp := 0.0
	if p.summaryCount > 0 {
		sp = math.Pow(pauseSummaryMultiplier, float64(p.summaryCount))
	}

	dp := 0.0
	if p.disinfoCount > 0 {
		dp = math.Pow(pauseDisinfoMultiplier, float64(p.disinfoCount))
	}

	if p.timeoutAt.IsZero() {
		p.timeoutAt = time.Now()
	}

	tp := startPauseTimeoutSeconds * (sp + dp)
	p.timeoutAt = p.timeoutAt.Add(time.Duration(tp) * time.Second)
	maxTimeoutAt := time.Now().Add(time.Duration(maxPauseTimeoutSeconds) * time.Second)
	if p.timeoutAt.After(maxTimeoutAt) {
		p.timeoutAt = maxTimeoutAt
	}

	logger.Debugf(e, "pausing %s in %s until %s (summary: %d, disinfo: %d)", e.From, e.ReplyTarget(), elapse.TimeDescription(p.timeoutAt), p.summaryCount, p.disinfoCount)
}

func parseURLFromMessage(message string) string {
	urlRegex := regexp.MustCompile(`(?i)(https?://\S+)`)
	urlMatches := urlRegex.FindStringSubmatch(message)
	if len(urlMatches) > 0 {
		return urlMatches[0]
	}
	return ""
}

var domainDenylist = []string{
	"i.redd.it",
}

func (c *SummaryCommand) isRootDomainIn(url string, domains []string) bool {
	root := domainutil.Domain(url)
	return slices.Contains(domains, root)
}

func (c *SummaryCommand) isDomainIn(url string, domains []string) bool {
	domain := retriever.Domain(url)
	return slices.Contains(domains, domain)
}

func (c *SummaryCommand) isRejectedTitle(title string) bool {
	for _, prefix := range c.cfg.Ignore.TitlePrefixes {
		if strings.HasPrefix(strings.ToLower(title), prefix) {
			return true
		}
	}
	return false
}

func (c *SummaryCommand) translatedURL(url string) (string, bool) {
	domain := strings.ToLower(domainutil.Domain(url))
	if translated, ok := c.cfg.Summary.TranslatedDomains[domain]; ok {
		return strings.Replace(url, domain, translated, 1), true
	}
	return url, false
}

func (c *SummaryCommand) actualURL(url string) (string, bool) {
	domain := strings.ToLower(domainutil.Domain(url))
	if avoidance, ok := c.cfg.Summary.AvoidanceDomains[domain]; ok {
		return strings.Replace(url, domain, avoidance, 1), true
	}
	return url, false
}
