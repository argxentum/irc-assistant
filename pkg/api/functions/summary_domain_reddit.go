package functions

import (
	"assistant/pkg/api/elapsed"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var redditCompleteDomainPattern = regexp.MustCompile(`https?://((?:.*?\.)?reddit\.com)/`)

func (f *summaryFunction) parseReddit(e *irc.Event, url string) (*summary, error) {
	if strings.Contains(url, "/s/") {
		return f.parseRedditShortlink(e, url)
	}

	logger := log.Logger()

	if reddit.IsJWTExpired(f.ctx.Session().Reddit.JWT) {
		logger.Debug(e, "reddit JWT token expired, logging in")
		result, err := reddit.Login(f.cfg.Reddit.Username, f.cfg.Reddit.Password)
		if err != nil {
			logger.Errorf(e, "error logging into reddit: %s", err)
			return nil, err
		}

		if result == nil {
			logger.Errorf(e, "unable to login to reddit")
			return nil, err
		}

		f.ctx.Session().Reddit.JWT = result.JWT
		f.ctx.Session().Reddit.Modhash = result.Modhash
		f.ctx.Session().Reddit.CookieJar.SetCookies(result.URL, result.Cookies)
	}

	match := redditCompleteDomainPattern.FindStringSubmatch(url)
	if len(match) < 2 {
		return nil, fmt.Errorf("unable to parse reddit domain from URL %s", url)
	}

	domain := match[1]
	url = strings.Replace(url, domain, "api.reddit.com", 1)

	logger.Debugf(e, "fetching reddit API URL %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", f.cfg.Reddit.UserAgent)

	client := &http.Client{
		Jar: f.ctx.Session().Reddit.CookieJar,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, errors.New("no response")
	}

	defer resp.Body.Close()

	var listings []reddit.Listing
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		return nil, err
	}

	if len(listings) == 0 {
		return nil, fmt.Errorf("no reddit parent found")
	}

	if len(listings[0].Data.Children) == 0 {
		return nil, fmt.Errorf("no posts found in reddit listing")
	}

	post := listings[0].Data.Children[0].Data
	createdAt := time.Unix(int64(post.Created), 0)
	return createSummary(fmt.Sprintf("%s (Posted %s in r/%s by u/%s • %s points and %s comments)", style.Bold(strings.TrimSpace(post.Title)), elapsed.ElapsedTimeDescription(createdAt), post.Subreddit, post.Author, text.DecorateNumberWithCommas(post.Score), text.DecorateNumberWithCommas(post.NumComments))), nil
}

func (f *summaryFunction) parseRedditShortlink(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "reddit shortlink request for %s", url)

	doc, err := f.docRetriever.RetrieveDocument(e, retriever.DefaultParams(url), retriever.DefaultTimeout)
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
		created = fmt.Sprintf(" %s", elapsed.ElapsedTimeDescription(createdAt))
	}

	return createSummary(fmt.Sprintf("%s (Posted%s in %s by u/%s • %s points and %s comments)", style.Bold(strings.TrimSpace(title)), created, subreddit, author, text.DecorateNumberWithCommas(score), text.DecorateNumberWithCommas(comments))), nil
}
