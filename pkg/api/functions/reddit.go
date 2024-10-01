package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapsed"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type redditFunction struct {
	FunctionStub
	subreddit string
	retriever retriever.DocumentRetriever
}

func NewRedditFunction(subreddit string, ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, fmt.Sprintf("r/%s", subreddit))
	if err != nil {
		return nil, err
	}

	return &redditFunction{
		FunctionStub: stub,
		subreddit:    subreddit,
		retriever:    retriever.NewDocumentRetriever(),
	}, nil
}

func (f *redditFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *redditFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	query := strings.Join(tokens[1:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] r/%s %s", e.From, e.ReplyTarget(), f.subreddit, query)

	if isRedditJWTExpired(f.ctx.Session().Reddit.JWT) {
		logger.Debug(e, "reddit JWT token expired, logging in")
		err := f.redditLogin()
		if err != nil {
			logger.Errorf(e, "error logging into reddit: %s", err)
			return
		}
	}

	posts, err := f.searchNewSubredditPosts(query)
	if err != nil {
		logger.Warningf(e, "unable to retrieve %s posts in r/%s: %s", query, f.subreddit, err)
		f.Replyf(e, "Unable to retrieve r/%s posts", f.subreddit)
		return
	}
	if len(posts) == 0 {
		logger.Warningf(e, "no %s posts in r/%s", query, f.subreddit)
		f.Replyf(e, "No r/%s posts found for %s", f.subreddit, style.Bold(query))
		return
	}
	f.sendPostMessages(e, posts)
}

const postsTitleMaxLength = 256

func (f *redditFunction) sendPostMessages(e *irc.Event, posts []RedditPost) {
	content := make([]string, 0)
	for i, post := range posts {
		title := post.Title
		if len(title) == 0 {
			continue
		}
		if len(title) > postsTitleMaxLength {
			title = title[:postsTitleMaxLength] + "..."
		}
		content = append(content, fmt.Sprintf("%s (r/%s, %s)", style.Bold(title), f.subreddit, elapsed.ElapsedTimeDescription(time.Unix(int64(post.Created), 0))))
		content = append(content, post.URL)
		if i < len(posts)-1 {
			content = append(content, " ")
		}
	}

	f.SendMessages(e, e.ReplyTarget(), content)
}

type RedditListing struct {
	Data struct {
		Children []struct {
			Data RedditPost
		}
	}
}

type RedditPost struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Created float64 `json:"created_utc"`
}

const topRedditPosts = "https://api.reddit.com/r/%s/top.json?limit=%d"
const searchRedditPosts = "https://api.reddit.com/r/%s/search.json?sort=new&limit=1&restrict_sr=on&q=title:%s"
const defaultRedditPosts = 3
const maxRedditPosts = 5

func topSubredditPosts(subreddit string, n int) ([]RedditPost, error) {
	if n == 0 {
		n = defaultRedditPosts
	} else if n > maxRedditPosts {
		n = maxRedditPosts
	}

	query := fmt.Sprintf(topRedditPosts, subreddit, n)
	resp, err := http.Get(query)
	if err != nil {
		return nil, err
	}

	var listing RedditListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	posts := make([]RedditPost, 0)
	for _, child := range listing.Data.Children {
		posts = append(posts, child.Data)
	}

	return posts, nil
}

func (f *redditFunction) searchNewSubredditPosts(topic string) ([]RedditPost, error) {
	t := url.QueryEscape(topic)
	query := fmt.Sprintf(searchRedditPosts, f.subreddit, t)

	req, err := http.NewRequest(http.MethodGet, query, nil)
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

	var listing RedditListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	posts := make([]RedditPost, 0)
	for _, child := range listing.Data.Children {
		posts = append(posts, child.Data)
	}

	return posts, nil
}

func (f *redditFunction) redditLogin() error {
	data := url.Values{}
	data.Set("user", f.cfg.Reddit.Username)
	data.Set("passwd", f.cfg.Reddit.Password)
	data.Set("api_type", "json")

	req, err := http.NewRequest(http.MethodPost, "https://ssl.reddit.com/api/login", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range retriever.RandomHeaderSet() {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
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
		return err
	}

	f.ctx.Session().Reddit.Modhash = body.JSON.Data.Modhash
	f.ctx.Session().Reddit.JWT = body.JSON.Data.Cookie
	u, err := url.Parse("https://reddit.com")
	if err != nil {
		return err
	}

	f.ctx.Session().Reddit.CookieJar.SetCookies(u, resp.Cookies())
	return nil
}

func isRedditJWTExpired(tok string) bool {
	if len(tok) == 0 {
		return true
	}

	token, _ := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) {
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
