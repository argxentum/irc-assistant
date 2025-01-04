package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"errors"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"
)

const SummaryCommandName = "summary"

const minimumTitleLength = 16
const maximumTitleLength = 256
const minimumPreferredTitleLength = 64
const maximumDescriptionLength = 300
const startRateLimitTimeoutSeconds = 30
const maxRateLimitTimeoutSeconds = 600
const rateLimitSummaryMultiplier = 1.3
const rateLimitDisinfoMultiplier = 2.0
const rateLimitShowWarningAfter = 2

type summary struct {
	messages []string
}

type UserRateLimit struct {
	channel      string
	nick         string
	summaryCount int
	disinfoCount int
	timeoutAt    time.Time
	ignoreCount  int
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

type SummaryCommand struct {
	*commandStub
	bodyRetriever  retriever.BodyRetriever
	docRetriever   retriever.DocumentRetriever
	userRateLimits map[string]*UserRateLimit
}

func NewSummaryCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &SummaryCommand{
		commandStub:    defaultCommandStub(ctx, cfg, irc),
		bodyRetriever:  retriever.NewBodyRetriever(),
		docRetriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
		userRateLimits: make(map[string]*UserRateLimit),
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
	rl := c.userRateLimits[e.From+"@"+e.ReplyTarget()]

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

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), url)

	if !e.IsPrivateMessage() && rl != nil {
		if rl.timeoutAt.After(time.Now()) {
			logger.Debugf(e, "ignoring rate limited request from %s in %s", e.From, e.ReplyTarget())
			dis := channel != nil && channel.Summarization.IsPossibleDisinformation(url)
			rl.ignoreCount++
			rl.summaryCount++
			if dis {
				rl.disinfoCount++
			}
			updateRateLimit(e, rl)
			if rl.ignoreCount > rateLimitShowWarningAfter {
				c.Replyf(e, "You've been rate limited. I'll resume summarizing your linked content in %s.", elapse.FutureTimeDescriptionConcise(rl.timeoutAt))
			}
			c.userRateLimits[e.From+"@"+e.ReplyTarget()] = rl
			return
		} else {
			logger.Debugf(e, "rate limit expired for %s in %s", e.From, e.ReplyTarget())
			rl.timeoutAt = time.Time{}
			rl.summaryCount = 0
			rl.disinfoCount = 0
			rl.ignoreCount = 0
		}
	}

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

			dis := false
			if channel != nil && channel.Summarization.IsPossibleDisinformation(url) {
				dis = true
				logger.Debugf(e, "URL is possible disinformation: %s", url)
				ds.messages = append(ds.messages, "⚠️ Possible disinformation, use caution.")
			}
			c.completeSummary(e, e.ReplyTarget(), ds.messages, dis, rl)
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
			c.completeSummary(e, e.ReplyTarget(), s.messages, false, rl)
			return
		}
	}

	s, err := c.summarize(e, url)
	if err != nil {
		logger.Debugf(e, "unable to summarize %s: %s", url, err)
	}

	if s == nil {
		logger.Debugf(e, "unable to summarize %s", url)
	} else {
		dis := false
		if channel != nil && channel.Summarization.IsPossibleDisinformation(url) {
			dis = true
			logger.Debugf(e, "URL is possible disinformation: %s", url)
			s.messages = append(s.messages, "⚠️ Possible disinformation, use caution.")
		}
		c.completeSummary(e, e.ReplyTarget(), s.messages, dis, rl)
	}
}

func (c *SummaryCommand) InitializeUserRateLimit(channel, nick string, duration time.Duration) *UserRateLimit {
	logger := log.Logger()

	rl := c.userRateLimits[nick+"@"+channel]
	if rl == nil || rl.timeoutAt.Before(time.Now()) {
		rl = &UserRateLimit{
			channel:     channel,
			nick:        nick,
			timeoutAt:   time.Now().Add(duration),
			ignoreCount: 0,
		}
		c.userRateLimits[nick+"@"+channel] = rl
		logger.Debugf(nil, "join, rate limiting %s in %s until %s", nick, channel, elapse.TimeDescription(rl.timeoutAt))
	} else {
		logger.Debugf(nil, "join, rate limit already in effect for %s in %s until %s", nick, channel, elapse.TimeDescription(rl.timeoutAt))
	}

	return rl
}

func (c *SummaryCommand) completeSummary(e *irc.Event, target string, messages []string, dis bool, rl *UserRateLimit) {
	if !e.IsPrivateMessage() {
		if rl == nil {
			rl = &UserRateLimit{
				channel: target,
				nick:    e.From,
			}
		}
		rl.summaryCount++
		if dis {
			rl.disinfoCount++
		}
		updateRateLimit(e, rl)
		c.userRateLimits[e.From+"@"+target] = rl
	}

	c.SendMessages(e, target, messages)
}

func updateRateLimit(e *irc.Event, rl *UserRateLimit) {
	logger := log.Logger()
	sp := 0.0
	if rl.summaryCount > 0 {
		sp = math.Pow(rateLimitSummaryMultiplier, float64(rl.summaryCount))
	}

	dp := 0.0
	if rl.disinfoCount > 0 {
		dp = math.Pow(rateLimitDisinfoMultiplier, float64(rl.disinfoCount))
	}

	if rl.timeoutAt.IsZero() {
		rl.timeoutAt = time.Now()
	}

	tp := startRateLimitTimeoutSeconds * (sp + dp)
	rl.timeoutAt = rl.timeoutAt.Add(time.Duration(tp) * time.Second)
	maxTimeoutAt := time.Now().Add(time.Duration(maxRateLimitTimeoutSeconds) * time.Second)
	if rl.timeoutAt.After(maxTimeoutAt) {
		rl.timeoutAt = maxTimeoutAt
	}

	logger.Debugf(e, "rate limiting %s in %s until %s (summary: %d, disinfo: %d)", e.From, e.ReplyTarget(), elapse.TimeDescription(rl.timeoutAt), rl.summaryCount, rl.disinfoCount)
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
	root := retriever.RootDomain(url)
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
