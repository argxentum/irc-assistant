package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"bytes"
	"errors"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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
const disinfoWarningMessage = "⚠️ Disinformation source, use caution"
const disinfoWarningMessageShort = "⚠️ Disinformation source"

type summary struct {
	messages []string
}

type UserPause struct {
	channel      string
	nick         string
	summaryCount int
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

type urlBundle struct {
	url      string
	original string
	actual   string
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

	original := parseURLFromMessage(e.Message())
	ub := urlBundle{url: original, original: original}

	if len(ub.url) == 0 {
		logger.Debugf(e, "no URL found in message")
		return
	}

	// some urls are used to avoid specific domains, e.g., xcancel.com to avoid x.com
	if actualURL, ok := c.actualURL(ub.url); ok {
		logger.Debugf(e, "actualURL %s to %s", ub.url, actualURL)
		ub.url = actualURL
	}

	ub.actual = ub.url

	// we need to translate some domains to get the actual content, e.g., x.com to fixupx.com
	if translatedURL, ok := c.translatedURL(ub.url); ok {
		logger.Debugf(e, "translatedURL %s to %s", ub.url, translatedURL)
		ub.url = translatedURL

		// ugly hack to parse out canonical url in translated urls
		if resp, _ := http.Get(ub.url); resp != nil {
			if data, _ := io.ReadAll(resp.Body); len(data) > 0 {
				if doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(data)); doc != nil {
					canonical := doc.Find(`link[rel="canonical"]`).First().AttrOr("href", "")
					if canonical != "" {
						ub.actual = canonical
						if translatedCanonicalURL, ok := c.translatedURL(canonical); ok {
							logger.Debugf(e, "canonicalURL %s to %s", canonical, translatedCanonicalURL)
							ub.url = translatedCanonicalURL
						}
					}
				}
			}
			defer resp.Body.Close()
		}
	}

	source, err := repository.FindSource(ub.url)
	if err != nil {
		logger.Errorf(nil, "error finding source, %s", err)
	}

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), ub.url)

	if c.isRootDomainIn(ub.url, c.cfg.Ignore.Domains) {
		logger.Debugf(e, "root domain denied %s", ub.url)
		return
	}

	if c.isDomainIn(ub.url, domainDenylist) {
		logger.Debugf(e, "domain denied %s", ub.url)
		return
	}

	dis := false
	if channel != nil && fs.IsDisinformationSource(channel.Name, ub.actual) {
		dis = true
		logger.Debugf(e, "URL is possible disinformation: %s", ub.url)
	}

	if c.requiresDomainSummary(ub.url) {
		logger.Debugf(e, "performing domain summarization for %s", ub.url)

		var ds *summary
		ds, source, err = c.domainSummary(e, ub.url)
		if err != nil {
			logger.Debugf(e, "domain specific summarization failed for %s: %s", ub.url, err)
		} else if ds != nil {
			logger.Debugf(e, "performed domain specific handling: %s", ub.url)
			c.completeSummary(e, source, ub, e.ReplyTarget(), ds.messages, dis, p)
		} else {
			logger.Debugf(e, "domain specific summarization failed for %s", ub.url)
		}
		return
	}

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(ub.url))
	if err != nil {
		logger.Debugf(e, "error retrieving document for %s: %v", ub.url, err)
		return
	}

	if !retriever.IsContentTypeAllowed(doc.Body.Response.Header.Get("Content-Type")) {
		logger.Debugf(e, "direct prefetch, disallowed content type for %s", ub.url)
		return
	}

	canonicalLink, _ := doc.Root.Find("link[rel='canonical']").First().Attr("href")
	if isValidCanonicalLink(ub.url, canonicalLink) {
		ub.actual = ub.url
		ub.url = canonicalLink
		doc, err = c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(ub.url))
		if err != nil {
			logger.Debugf(e, "error retrieving canonical document for %s: %v", ub.url, err)
			return
		}
	}

	if !e.IsPrivateMessage() && p != nil {
		if p.timeoutAt.After(time.Now()) {
			logger.Debugf(e, "ignoring paused summary request from %s in %s", e.From, e.ReplyTarget())
			if dis {
				c.addDisinformationPenalty(e, 1)
				c.SendMessage(e, e.ReplyTarget(), disinfoWarningMessage)
			}

			cn := c.findCommunityNotes(e, ub.url)
			if len(cn) > 0 {
				c.SendMessages(e, e.ReplyTarget(), cn)
			}

			p.ignoreCount++
			p.summaryCount++
			updatePause(e, p)
			c.userPauses[e.From+"@"+e.ReplyTarget()] = p
			return
		} else {
			logger.Debugf(e, "pause expired for %s in %s", e.From, e.ReplyTarget())
			p.timeoutAt = time.Time{}
			p.summaryCount = 0
			p.ignoreCount = 0
		}
	}

	contentSummarizer, err := c.contentSummary(e, doc)
	if err != nil {
		if errors.Is(err, retriever.DisallowedContentTypeError) {
			logger.Debugf(e, "disallowed content type for %s", ub.url)
			return
		}
	}

	if contentSummarizer != nil {
		logger.Debugf(e, "performing content summarization for %s", ub.url)

		s, err := contentSummarizer(e, doc)
		if err != nil {
			logger.Debugf(e, "content specific summarization failed for %s: %s", ub.url, err)
		}
		if s != nil {
			messages := s.messages

			if source != nil {
				messages = append(messages, repository.ShortSourceSummary(source))
			}

			c.completeSummary(e, source, ub, e.ReplyTarget(), messages, dis, p)
			return
		}
	}

	s, err := c.summarize(e, doc)
	if err != nil {
		logger.Debugf(e, "unable to summarize %s: %s", ub.url, err)
	}

	if s == nil {
		logger.Debugf(e, "unable to summarize %s", ub.url)
	} else {
		c.completeSummary(e, source, ub, e.ReplyTarget(), s.messages, dis, p)
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

func (c *SummaryCommand) completeSummary(e *irc.Event, source *models.Source, ub urlBundle, target string, messages []string, dis bool, p *UserPause) {
	logger := log.Logger()

	if !e.IsPrivateMessage() {
		if p == nil {
			p = &UserPause{
				channel: target,
				nick:    e.From,
			}
		}
		p.summaryCount++
		updatePause(e, p)
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

	if e.Metadata != nil {
		logger.Debugf(e, "event has metadata: %v", e.Metadata)

		if showURL, ok := e.Metadata[CommandMetadataShowURL]; ok {
			if showURL.(bool) {
				unescapedMessages = append(unescapedMessages, ub.url)
			}
		}
	}

	sourceSummary := c.combinedSourceSummary(e, ub, source, dis)
	if len(sourceSummary) > 0 {
		unescapedMessages = append(unescapedMessages, sourceSummary)
	}

	cn := c.findCommunityNotes(e, ub.url)
	if len(cn) > 0 {
		logger.Debugf(e, "adding community notes to output")
		unescapedMessages = append(unescapedMessages, cn...)
	}

	c.SendMessages(e, target, unescapedMessages)
}

func (c *SummaryCommand) combinedSourceSummary(e *irc.Event, ub urlBundle, source *models.Source, dis bool) string {
	logger := log.Logger()
	sourceSummary := ""

	if dis {
		logger.Debugf(e, "content is possible disinformation, applying penalty and adding warning")
		c.addDisinformationPenalty(e, 1)
	}

	if source != nil {
		logger.Debugf(e, "adding source details to output")
		sourceSummary += repository.ShortSourceSummary(source)

		if dis {
			logger.Debugf(e, "adding short disinformation warning message to output")
			if len(sourceSummary) > 0 {
				sourceSummary += " | "
			}
			sourceSummary += disinfoWarningMessageShort
		}

		if source.Paywall {
			var id string
			var err error
			if c.isRootDomainIn(ub.actual, source.URLs) {
				id, err = repository.GetArchiveShortcutID(ub.actual)
			} else if c.isRootDomainIn(ub.url, source.URLs) {
				id, err = repository.GetArchiveShortcutID(ub.url)
			}

			if err == nil && len(id) > 0 {
				logger.Debugf(e, "adding paywall avoidance url to output")
				if len(sourceSummary) > 0 {
					sourceSummary += " | "
				}
				sourceSummary += "\U0001F513 " + fmt.Sprintf(shortcutURLPattern, c.cfg.Web.ExternalRootURL) + id
			}
		}
	} else if dis {
		logger.Debugf(e, "adding long disinformation warning message to output")
		sourceSummary = disinfoWarningMessage
	}

	return sourceSummary
}

func updatePause(e *irc.Event, p *UserPause) {
	logger := log.Logger()
	sp := 0.0
	if p.summaryCount > 0 {
		sp = math.Pow(pauseSummaryMultiplier, float64(p.summaryCount))
	}

	dp := 0.0

	if p.timeoutAt.IsZero() {
		p.timeoutAt = time.Now()
	}

	tp := startPauseTimeoutSeconds * (sp + dp)
	p.timeoutAt = p.timeoutAt.Add(time.Duration(tp) * time.Second)
	maxTimeoutAt := time.Now().Add(time.Duration(maxPauseTimeoutSeconds) * time.Second)
	if p.timeoutAt.After(maxTimeoutAt) {
		p.timeoutAt = maxTimeoutAt
	}

	logger.Debugf(e, "pausing %s in %s until %s (summary: %d)", e.From, e.ReplyTarget(), elapse.TimeDescription(p.timeoutAt), p.summaryCount)
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

func (c *SummaryCommand) addDisinformationPenalty(e *irc.Event, penalty int) {
	logger := log.Logger()
	logger.Debugf(e, "incrementing disinformation penalty for %s in %s by %d", e.From, e.ReplyTarget(), penalty)

	u, err := repository.GetUserByNick(e, e.ReplyTarget(), e.From, false)
	if err != nil {
		logger.Errorf(e, "error getting user by nick: %v", err)
		return
	}

	if u == nil {
		logger.Errorf(e, "user %s not found in %s for disinformation penalty removal request", e.From, e.ReplyTarget())
		return
	}

	u.Penalty += penalty
	if u.Penalty < 0 {
		u.Penalty = 0
	}

	fs := firestore.Get()
	err = fs.UpdateUser(e.ReplyTarget(), u, map[string]any{"penalty": u.Penalty, "updated_at": time.Now()})
	if err != nil {
		logger.Errorf(e, "error updating penalty for %s: %v", e.ReplyTarget(), err)
		return
	}

	logger.Debug(e, "adding disinformation penalty removal task")

	task := models.NewDisinformationPenaltyRemovalTask(time.Now().Add(time.Duration(c.cfg.DisinfoPenalty.TimeoutSeconds)*time.Second), e.ReplyTarget(), e.From, penalty)
	err = firestore.Get().AddTask(task)
	if err != nil {
		logger.Errorf(e, "error adding disinformation penalty removal task, %s", err)
		return
	}

	if u.Penalty >= c.cfg.DisinfoPenalty.Threshold {
		c.ExecuteSynthesizedEvent(e, MuteCommandName, fmt.Sprintf("%s %s disinformation threshold reached", c.cfg.DisinfoPenalty.Duration, e.From), nil)
	}
}

func (c *SummaryCommand) findCommunityNotes(e *irc.Event, url string) []string {
	if e.IsPrivateMessage() {
		return nil
	}

	logger := log.Logger()

	note, err := repository.GetCommunityNoteForSource(e, e.ReplyTarget(), url)
	if err != nil {
		logger.Errorf(e, "error getting community note for %s: %v", url, err)
		return nil
	}

	if note == nil {
		return nil
	}

	logger.Debugf(e, "adding community note %s for %s", note.ID, url)

	includeCounterSourceURL := !slices.Contains(note.CounterSources, url)
	return createCommunityNoteOutputMessages(e, note, includeCounterSourceURL)
}

func (c *SummaryCommand) createSummaryFromTitleAndDescription(title, description string) (*summary, error) {
	if len(title) > maximumTitleLength {
		title = title[:maximumTitleLength] + "..."
	}

	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	if c.isRejectedTitle(title) {
		return nil, rejectedTitleError
	}

	if len(title)+len(description) < minimumTitleLength {
		return nil, summaryTooShortError
	}

	if len(title) > 0 && len(description) > 0 {
		if text.MostlyContains(title, description, 0.9) {
			if len(description) > len(title) {
				return createSummary(style.Bold(description)), nil
			}
			return createSummary(style.Bold(title)), nil
		}
		return createSummary(fmt.Sprintf("%s%s %s", style.Bold(title), getSummaryFieldSeparator(title), description)), nil
	}

	if len(title) > 0 {
		return createSummary(style.Bold(title)), nil
	}

	if len(description) > 0 {
		return createSummary(style.Bold(description)), nil
	}

	return nil, noContentError
}

func getSummaryFieldSeparator(title string) string {
	end := title[len(title)-1]
	if end == '.' || end == '!' || end == '?' {
		return ""
	}
	return ":"
}
