package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var redditCompleteDomainPattern = regexp.MustCompile(`https?://((?:.*?\.)?reddit\.com)/`)

func (c *SummaryCommand) parseReddit(e *irc.Event, url string) (*summary, error) {
	if strings.Contains(url, "/s/") {
		return c.parseRedditShortlink(e, url)
	}

	if strings.HasPrefix(url, "https://old.reddit.com/") {
		url = strings.Replace(url, "old.reddit.com", "reddit.com", 1)
	}

	logger := log.Logger()

	if reddit.IsJWTExpired(c.ctx.Session().Reddit.JWT) {
		logger.Debug(e, "reddit JWT token expired, logging in")
		result, err := reddit.Login(c.cfg.Reddit.Username, c.cfg.Reddit.Password)
		if err != nil {
			logger.Errorf(e, "error logging into reddit: %s", err)
			return nil, err
		}

		if result == nil {
			logger.Errorf(e, "unable to login to reddit")
			return nil, err
		}

		c.ctx.Session().Reddit.JWT = result.JWT
		c.ctx.Session().Reddit.Modhash = result.Modhash
		c.ctx.Session().Reddit.CookieJar.SetCookies(result.URL, result.Cookies)
	}

	match := redditCompleteDomainPattern.FindStringSubmatch(url)
	if len(match) < 2 {
		return nil, fmt.Errorf("unable to parse reddit domain from URL %s", url)
	}

	domain := match[1]
	url = strings.Replace(url, domain, "api.reddit.com", 1)

	post, err := reddit.GetPostWithTopComment(c.ctx, c.cfg, url)
	if err != nil {
		return nil, err
	}

	if post == nil {
		return nil, errors.New("post not found")
	}

	title := text.SanitizeSummaryContent(post.Post.Title)
	if len(title) == 0 {
		return nil, nil
	}

	messages := make([]string, 0)
	messages = append(messages, post.Post.FormattedTitle())

	if post.Comment != nil {
		messages = append(messages, post.Comment.FormattedBody())
	}

	source, err := repository.FindSource(post.Post.URL)
	if err != nil {
		logger.Errorf(nil, "error finding source, %s", err)
	}

	if source != nil {
		messages = append(messages, repository.ShortSourceSummary(source))
	}

	return createSummary(messages...), nil
}

func (c *SummaryCommand) parseRedditShortlink(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "reddit shortlink request for %s", url)

	doc, err := c.docRetriever.RetrieveDocument(e, retriever.DefaultParams(url), retriever.DefaultTimeout)
	if err != nil {
		logger.Debugf(e, "unable to retrieve reddit shortlink for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve reddit shortlink for %s", url)
		return nil, fmt.Errorf("reddit shortlink doc nil")
	}

	post := doc.Find("shreddit-post").First()
	title := strings.TrimSpace(post.AttrOr("post-title", ""))
	author := strings.TrimSpace(post.AttrOr("author", ""))
	link := strings.TrimSpace(post.AttrOr("content-href", ""))
	subreddit := strings.TrimSpace(post.AttrOr("subreddit-prefixed-name", ""))
	created := strings.TrimSpace(post.AttrOr("created-timestamp", ""))

	var createdAt time.Time
	if len(created) > 0 {
		createdAt, err = time.Parse("2006-01-02T15:04:05+0000", created)
		if err != nil {
			logger.Debugf(e, "unable to parse created timestamp: %s", err)
		}
	}

	comments := 0
	comments, err = strconv.Atoi(strings.TrimSpace(post.AttrOr("comment-count", "")))
	if err != nil {
		logger.Debugf(e, "unable to parse comment count: %s", err)
	}

	score := 0
	score, err = strconv.Atoi(strings.TrimSpace(post.AttrOr("score", "")))
	if err != nil {
		logger.Debugf(e, "unable to parse score: %s", err)
	}

	if len(title) == 0 {
		logger.Debugf(e, "reddit shortlink title empty")
		return nil, summaryTooShortError
	}

	if len(link) == 0 {
		logger.Debugf(e, "reddit shortlink link empty")
		return nil, summaryTooShortError
	}

	created = ""
	if !createdAt.IsZero() {
		created = fmt.Sprintf(" %s", elapse.TimeDescription(createdAt))
	}

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s (Posted%s in %s by u/%s • %s points and %s comments)", style.Bold(strings.TrimSpace(title)), created, subreddit, author, text.DecorateNumberWithCommas(score), text.DecorateNumberWithCommas(comments)))

	source, err := repository.FindSource(link)
	if err != nil {
		logger.Errorf(nil, "error finding source, %s", err)
	}

	if source != nil {
		messages = append(messages, repository.ShortSourceSummary(source))
	}

	return createSummary(messages...), nil
}
