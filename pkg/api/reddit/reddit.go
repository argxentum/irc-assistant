package reddit

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/retriever"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type LoginResult struct {
	Modhash string
	JWT     string
	URL     *url.URL
	Cookies []*http.Cookie
}

type Listing struct {
	Data struct {
		Children []struct {
			Data Post
		}
	}
}

type Post struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Permalink   string  `json:"permalink"`
	Created     float64 `json:"created_utc"`
	Subreddit   string  `json:"subreddit"`
	Author      string  `json:"author"`
	Score       int     `json:"score"`
	NumComments int     `json:"num_comments"`
	Stickied    bool    `json:"stickied"`
}

type PostDetail []struct {
	Data struct {
		Children []struct {
			Data Comment
		}
	}
}

type PostWithTopComment struct {
	Post    Post
	Comment *Comment
}

type Comment struct {
	Body   string `json:"body"`
	Author string `json:"author"`
}

func IsJWTExpired(tok string) bool {
	if len(tok) == 0 {
		return true
	}

	token, _ := jwt.Parse(tok, func(token *jwt.Token) (any, error) {
		return token, nil
	})

	if token == nil {
		return true
	}

	exp, err := token.Claims.GetExpirationTime()
	if err != nil || exp == nil {
		return true
	}

	return time.Now().Unix() > exp.Unix()
}

func Login(username, password string) (*LoginResult, error) {
	data := url.Values{}
	data.Set("user", username)
	data.Set("passwd", password)
	data.Set("api_type", "json")

	req, err := http.NewRequest(http.MethodPost, "https://ssl.reddit.com/api/login", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range retriever.RandomHeaderSet() {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, errors.New("no response")
	}

	var body struct {
		JSON struct {
			Errors []string
			Data   struct {
				Modhash string
				Cookie  string
			}
		}
	}

	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	result := &LoginResult{}

	result.Modhash = body.JSON.Data.Modhash
	result.JWT = body.JSON.Data.Cookie
	u, err := url.Parse("https://reddit.com")
	if err != nil {
		return nil, err
	}

	result.URL = u
	result.Cookies = resp.Cookies()
	return result, nil
}

func loginIfNeeded(ctx context.Context, cfg *config.Config) error {
	logger := log.Logger()

	if IsJWTExpired(ctx.Session().Reddit.JWT) {
		logger.Debug(nil, "reddit JWT token expired, logging in")

		result, err := Login(cfg.Reddit.Username, cfg.Reddit.Password)
		if err != nil {
			return fmt.Errorf("error logging into reddit, %s", err)
		}

		if result == nil {
			return errors.New("unable to login to reddit")
		}

		ctx.Session().Reddit.JWT = result.JWT
		ctx.Session().Reddit.Modhash = result.Modhash
		ctx.Session().Reddit.CookieJar.SetCookies(result.URL, result.Cookies)
	}

	return nil
}

const redditBaseURL = "https://api.reddit.com"
const searchSubredditPosts = "%s/r/%s/search.json?sort=new&limit=1&restrict_sr=on&q=title:%s"
const subredditCategoryPosts = "%s/r/%s/%s.json?limit=%d"
const defaultRedditPosts = 3
const maxRedditPosts = 5

func SearchNewSubredditPosts(ctx context.Context, cfg *config.Config, subreddit, topic string) ([]PostWithTopComment, error) {
	logger := log.Logger()
	if err := loginIfNeeded(ctx, cfg); err != nil {
		return nil, err
	}

	t := url.QueryEscape(topic)
	query := fmt.Sprintf(searchSubredditPosts, redditBaseURL, subreddit, t)

	req, err := http.NewRequest(http.MethodGet, query, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", cfg.Reddit.UserAgent)

	client := &http.Client{
		Jar: ctx.Session().Reddit.CookieJar,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var listing Listing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	posts := make([]PostWithTopComment, 0)
	for _, child := range listing.Data.Children {
		post := child.Data
		comment, err := getTopComment(ctx, post.Permalink)
		if err != nil {
			logger.Warningf(nil, "error getting top comment for %s, %s", post.Permalink, err)
			continue
		}
		posts = append(posts, PostWithTopComment{Post: post, Comment: comment})
	}

	return posts, nil
}

func SubredditCategoryPostsWithTopComment(ctx context.Context, cfg *config.Config, subreddit, category string, n int) ([]PostWithTopComment, error) {
	if err := loginIfNeeded(ctx, cfg); err != nil {
		return nil, err
	}

	if n == 0 {
		n = defaultRedditPosts
	} else if n > maxRedditPosts {
		n = maxRedditPosts
	}

	query := fmt.Sprintf(subredditCategoryPosts, redditBaseURL, subreddit, category, n)
	req, err := http.NewRequest(http.MethodGet, query, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", cfg.Reddit.UserAgent)

	client := &http.Client{
		Jar: ctx.Session().Reddit.CookieJar,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	var listing Listing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	posts := make([]PostWithTopComment, 0)

	for _, child := range listing.Data.Children {
		if len(posts) >= n {
			break
		}

		post := child.Data

		if post.Stickied {
			continue
		}

		comment, err := getTopComment(ctx, post.Permalink)
		if err != nil {
			return nil, err
		}

		posts = append(posts, PostWithTopComment{Post: post, Comment: comment})
	}

	return posts, nil
}

func getTopComment(ctx context.Context, permalink string) (*Comment, error) {
	client := &http.Client{
		Jar: ctx.Session().Reddit.CookieJar,
	}

	u := redditBaseURL + html.UnescapeString(permalink)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	var detail PostDetail
	if err = json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if len(detail) < 2 || len(detail[1].Data.Children) == 0 || (len(detail[1].Data.Children) == 1 && detail[1].Data.Children[0].Data.Author == "AutoModerator") {
		return nil, nil
	}

	for _, comment := range detail[1].Data.Children {
		if comment.Data.Author != "AutoModerator" && len(comment.Data.Body) > 0 {
			return &comment.Data, nil
		}
	}

	return nil, nil
}
