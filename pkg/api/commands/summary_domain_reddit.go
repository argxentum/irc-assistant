package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"bytes"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const redditWebURL = "https://www.reddit.com"

var redditCompleteDomainPattern = regexp.MustCompile(`https?://((?:.*?\.)?reddit\.com)/`)
var redditMediaPattern = regexp.MustCompile(`https://(?:www\.)?reddit\.com/media\?url=https.+`)

func (c *SummaryCommand) parseReddit(e *irc.Event, url string) (*summary, error) {
	if strings.Contains(url, "/s/") {
		return c.parseRedditShortlink(e, url)
	}

	if redditMediaPattern.MatchString(url) {
		return c.parseRedditMediaLink(e, url)
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

	logger.Infof(e, "reddit media request for %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.cfg.Reddit.UserAgent)

	client := &http.Client{
		Jar: c.ctx.Session().Reddit.CookieJar,
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Debugf(nil, "error fetching %s, %s", url, err)
		return nil, err
	}

	if resp == nil {
		return nil, errors.New("no response")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Debugf(e, "unable to read reddit media link for %s: %s", url, err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		logger.Debugf(e, "unable to retrieve reddit media link for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve reddit shortlink for %s", url)
		return nil, fmt.Errorf("reddit shortlink doc nil")
	}

	// <shreddit-post permalink="/r/Weird/comments/1kqbrm3/my_right_hand_randomly_turned_orangish_brown_in/" ... >
	post := doc.Find("shreddit-post").First()
	permalink := strings.TrimSpace(post.AttrOr("permalink", ""))

	return c.parseReddit(e, redditWebURL+permalink)
}

func (c *SummaryCommand) parseRedditMediaLink(e *irc.Event, url string) (*summary, error) {
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

	logger.Infof(e, "reddit media request for %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.cfg.Reddit.UserAgent)

	client := &http.Client{
		Jar: c.ctx.Session().Reddit.CookieJar,
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Debugf(nil, "error fetching %s, %s", url, err)
		return nil, err
	}

	if resp == nil {
		return nil, errors.New("no response")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Debugf(e, "unable to read reddit media link for %s: %s", url, err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		logger.Debugf(e, "unable to retrieve reddit media link for %s: %s", url, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve reddit media link for %s", url)
		return nil, fmt.Errorf("reddit media link doc nil")
	}

	// <post-bottom-bar permalink="/r/funny/comments/1kpzega/those_rules_seem_awfully_broad/" ...>
	bottomBar := doc.Find("post-bottom-bar").First()
	permalink := strings.TrimSpace(bottomBar.AttrOr("permalink", ""))

	updatedUrl := redditWebURL + permalink
	s, err := c.parseReddit(e, updatedUrl)
	if s != nil {
		s.addMessage(updatedUrl)
	}

	return s, err
}
