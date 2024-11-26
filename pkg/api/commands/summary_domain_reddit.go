package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
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

func (c *summaryCommand) parseReddit(e *irc.Event, url string) (*summary, error) {
	if strings.Contains(url, "/s/") {
		return c.parseRedditShortlink(e, url)
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

	title := text.Sanitize(post.Post.Title)
	if len(title) == 0 {
		return nil, nil
	}

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s (r/%s, %s)", style.Bold(title), post.Post.Subreddit, elapse.TimeDescription(time.Unix(int64(post.Post.Created), 0))))

	if post.Comment != nil {
		comment := text.Sanitize(post.Comment.Body)
		messages = append(messages, fmt.Sprintf("Top comment (by u/%s): %s", post.Comment.Author, style.Italics(comment)))
	}

	return createSummary(messages...), nil
}

func (c *summaryCommand) parseRedditShortlink(e *irc.Event, url string) (*summary, error) {
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

	return createSummary(fmt.Sprintf("%s (Posted%s in %s by u/%s • %s points and %s comments)", style.Bold(strings.TrimSpace(title)), created, subreddit, author, text.DecorateNumberWithCommas(score), text.DecorateNumberWithCommas(comments))), nil
}
