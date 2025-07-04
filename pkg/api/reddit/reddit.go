package reddit

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/marshaling"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"html"
	"net/http"
	"net/url"
	"slices"
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

func (p Post) FormattedTitle() string {
	title := html.UnescapeString(text.SanitizeSummaryContent(p.Title))
	return fmt.Sprintf("%s (r/%s, %s)", style.Bold(title), p.Subreddit, elapse.TimeDescription(time.Unix(int64(p.Created), 0)))
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
	Body          string `json:"body"`
	Author        string `json:"author"`
	Distinguished string `json:"distinguished"`
}

func (c *Comment) IsFromModerator() bool {
	return strings.ToLower(c.Distinguished) == "moderator"
}

func (c *Comment) FormattedBody() string {
	comment := html.UnescapeString(text.SanitizeSummaryContent(c.Body))

	if c.Author == "[deleted]" {
		return fmt.Sprintf("Top comment: %s", style.Italics(text.SanitizeSummaryContent(comment)))
	} else {
		return fmt.Sprintf("Top comment, by u/%s: %s", c.Author, style.Italics(text.SanitizeSummaryContent(comment)))
	}
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

func Login(ctx context.Context, cfg *config.Config) error {
	logger := log.Logger()

	if IsJWTExpired(ctx.Session().Reddit.JWT) {
		logger.Debug(nil, "reddit JWT token expired, logging in")

		result, err := login(cfg.Reddit.Username, cfg.Reddit.Password)
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

func login(username, password string) (*LoginResult, error) {
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

const redditBaseURL = "https://api.reddit.com"
const searchNewSubredditPosts = "%s/r/%s/search.json?sort=new&limit=1&restrict_sr=on&q=title:%s"
const searchRelevantSubredditPosts = "%s/r/%s/search.json?sort=relevance&t=all&limit=1&restrict_sr=on&q=%s"
const searchPostsForURL = "%s/search.json?limit=1&restrict_sr=on&q=url:%s"
const subredditCategoryPosts = "%s/r/%s/%s.json?limit=%d"
const defaultRedditPosts = 3
const maxRedditPosts = 5

func SearchNewSubredditPosts(ctx context.Context, cfg *config.Config, subreddit, topic string) ([]PostWithTopComment, error) {
	logger := log.Logger()
	if err := Login(ctx, cfg); err != nil {
		return nil, err
	}
	t := url.QueryEscape(topic)
	u := fmt.Sprintf(searchNewSubredditPosts, redditBaseURL, subreddit, t)
	logger.Debugf(nil, "reddit new search URL: %s", u)
	return searchSubredditPosts(ctx, cfg, u)
}

func SearchRelevantSubredditPosts(ctx context.Context, cfg *config.Config, subreddit, topic string) ([]PostWithTopComment, error) {
	logger := log.Logger()
	if err := Login(ctx, cfg); err != nil {
		return nil, err
	}
	t := url.QueryEscape(topic)
	u := fmt.Sprintf(searchRelevantSubredditPosts, redditBaseURL, subreddit, t)
	logger.Debugf(nil, "reddit relevant search URL: %s", u)
	return searchSubredditPosts(ctx, cfg, u)
}

func searchSubredditPosts(ctx context.Context, cfg *config.Config, u string) ([]PostWithTopComment, error) {
	logger := log.Logger()
	if err := Login(ctx, cfg); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
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
	if err := marshaling.Unmarshal(resp.Body, &listing); err != nil {
		return nil, err
	}

	posts := make([]PostWithTopComment, 0)
	for _, child := range listing.Data.Children {
		post := child.Data
		comment, err := getTopComment(ctx, cfg, post.Permalink)
		if err != nil {
			logger.Warningf(nil, "error getting top comment for %s, %s", post.Permalink, err)
		}
		posts = append(posts, PostWithTopComment{Post: post, Comment: comment})
	}

	return posts, nil
}

func SearchPostsForURL(ctx context.Context, cfg *config.Config, bodyURL string) ([]PostWithTopComment, error) {
	logger := log.Logger()
	if err := Login(ctx, cfg); err != nil {
		return nil, err
	}

	t := url.QueryEscape(bodyURL)
	query := fmt.Sprintf(searchPostsForURL, redditBaseURL, t)

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
	if err := marshaling.Unmarshal(resp.Body, &listing); err != nil {
		return nil, err
	}

	if len(listing.Data.Children) == 0 {
		return nil, errors.New("no posts found")
	}

	posts := make([]PostWithTopComment, 0)
	for _, child := range listing.Data.Children {
		post := child.Data

		if !slices.ContainsFunc(cfg.Reddit.SummarizationSubreddits, func(s string) bool {
			return strings.ToLower(s) == strings.ToLower(post.Subreddit)
		}) {
			logger.Debugf(nil, "skipping post from %s", post.Subreddit)
			continue
		}

		comment, err := getTopComment(ctx, cfg, post.Permalink)
		if err != nil {
			logger.Warningf(nil, "error getting top comment for %s, %s", post.Permalink, err)
		}
		posts = append(posts, PostWithTopComment{Post: post, Comment: comment})
	}

	return posts, nil
}

func SubredditCategoryPostsWithTopComment(ctx context.Context, cfg *config.Config, subreddit, category string, n int) ([]PostWithTopComment, error) {
	logger := log.Logger()
	if err := Login(ctx, cfg); err != nil {
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
		logger.Errorf(nil, "error fetching reddit posts, %s", err)
		return nil, err
	}

	var listing Listing
	if err := marshaling.Unmarshal(resp.Body, &listing); err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	posts := make([]PostWithTopComment, 0)

	for _, child := range listing.Data.Children {
		if len(posts) >= n {
			logger.Debugf(nil, "reached max reddit posts")
			break
		}

		post := child.Data

		if post.Stickied {
			logger.Debugf(nil, "skipping stickied post %s", post.Title)
			continue
		}

		comment, err := getTopComment(ctx, cfg, post.Permalink)
		if err != nil {
			logger.Warningf(nil, "error getting top comment for %s, %s", post.Permalink, err)
		}

		posts = append(posts, PostWithTopComment{Post: post, Comment: comment})
	}

	return posts, nil
}

func GetPostWithTopComment(ctx context.Context, cfg *config.Config, apiURL string) (*PostWithTopComment, error) {
	if err := Login(ctx, cfg); err != nil {
		return nil, err
	}

	logger := log.Logger()
	logger.Debugf(nil, "fetching reddit API URL %s", apiURL)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", cfg.Reddit.UserAgent)

	client := &http.Client{
		Jar: ctx.Session().Reddit.CookieJar,
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Debugf(nil, "error fetching %s, %s", apiURL, err)
		return nil, err
	}

	if resp == nil {
		return nil, errors.New("no response")
	}

	defer resp.Body.Close()

	var listings []Listing
	if err := marshaling.Unmarshal(resp.Body, &listings); err != nil {
		return nil, err
	}

	if len(listings) == 0 {
		return nil, fmt.Errorf("no reddit parent found")
	}

	if len(listings[0].Data.Children) == 0 {
		return nil, fmt.Errorf("no posts found in reddit listing")
	}

	post := listings[0].Data.Children[0].Data
	comment, err := getTopComment(ctx, cfg, post.Permalink)
	if err != nil {
		logger.Warningf(nil, "error getting top comment for %s, %s", post.Permalink, err)
	}

	return &PostWithTopComment{Post: post, Comment: comment}, nil
}

func getTopComment(ctx context.Context, cfg *config.Config, permalink string) (*Comment, error) {
	if err := Login(ctx, cfg); err != nil {
		return nil, err
	}

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
	if err := marshaling.Unmarshal(resp.Body, &detail); err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if len(detail) < 2 || len(detail[1].Data.Children) == 0 || (len(detail[1].Data.Children) == 1 && (detail[1].Data.Children[0].Data.Author == "AutoModerator" || detail[1].Data.Children[0].Data.Body == "[deleted]")) {
		return nil, nil
	}

	for _, comment := range detail[1].Data.Children {
		if comment.Data.Author != "AutoModerator" && !comment.Data.IsFromModerator() && comment.Data.Body != "[deleted]" && len(comment.Data.Body) > 0 {
			return &comment.Data, nil
		}
	}

	return nil, nil
}
