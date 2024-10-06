package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var redditCompleteDomainPattern = regexp.MustCompile(`https?://((?:.*?\.)?reddit\.com)/`)

func (f *summaryFunction) parseRedditMessage(e *irc.Event, url string) (*summary, error) {
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
	return &summary{
		fmt.Sprintf("%s (Posted in r/%s by u/%s • %s points and %s comments)", style.Bold(strings.TrimSpace(post.Title)), post.Subreddit, post.Author, text.DecorateNumberWithCommas(post.Score), text.DecorateNumberWithCommas(post.NumComments)),
	}, nil
}
