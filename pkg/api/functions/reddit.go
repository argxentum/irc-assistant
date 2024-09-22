package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
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
}

func NewRedditFunction(subreddit string, ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, fmt.Sprintf("r/%s", subreddit))
	if err != nil {
		return nil, err
	}

	return &redditFunction{
		FunctionStub: stub,
		subreddit:    subreddit,
	}, nil
}

func (f *redditFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *redditFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ r/%s\n", f.subreddit)
	tokens := Tokens(e.Message())
	query := strings.Join(tokens[1:], " ")

	if isRedditJWTExpired(f.ctx.RedditJWT()) {
		err := f.redditLogin()
		if err != nil {
			fmt.Printf("error logging into reddit: %s\n", err)
			return
		}
	}

	posts, err := f.searchNewSubredditPosts(query)
	if err != nil {
		f.Reply(e, "Unable to retrieve r/%s posts", f.subreddit)
		return
	}
	if len(posts) == 0 {
		f.Reply(e, "No r/%s posts found for %s", f.subreddit, text.Bold(query))
		return
	}
	f.sendPostMessages(e, posts)
}

const postsTitleMaxLength = 256

func elapsedTimeDescription(t time.Time) string {
	elapsed := time.Now().Sub(t)

	year := time.Hour * 24 * 365
	month := time.Hour * 24 * 30
	week := time.Hour * 24 * 7
	day := time.Hour * 24
	hour := time.Hour
	minute := time.Minute
	second := time.Second

	if elapsed >= year {
		years := elapsed / year
		if years == 1 {
			return "last year"
		}
		return fmt.Sprintf("%d years ago", years)
	} else if elapsed >= month {
		months := elapsed / month
		if months == 1 {
			return "last month"
		}
		return fmt.Sprintf("%d months ago", months)
	} else if elapsed >= week {
		weeks := elapsed / week
		if weeks == 1 {
			return "last week"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if elapsed >= day {
		days := elapsed / day
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if elapsed >= hour {
		hours := elapsed / hour
		if hours == 1 {
			return "an hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if elapsed >= minute {
		minutes := elapsed / minute
		if minutes == 1 {
			return "a minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else {
		seconds := elapsed / second
		if seconds < 5 {
			return "just now"
		}
		if seconds < 30 {
			return "a few seconds ago"
		}
		return fmt.Sprintf("%d seconds ago", seconds)
	}
}

func (f *redditFunction) sendPostMessages(e *core.Event, posts []RedditPost) {
	content := make([]string, 0)
	for i, post := range posts {
		title := post.Title
		if len(title) == 0 {
			continue
		}
		if len(title) > postsTitleMaxLength {
			title = title[:postsTitleMaxLength] + "..."
		}
		content = append(content, fmt.Sprintf("%s (r/%s, %s)", text.Bold(title), f.subreddit, elapsedTimeDescription(time.Unix(int64(post.Created), 0))))
		content = append(content, post.URL)
		if i < len(posts)-1 {
			content = append(content, " ")
		}
	}

	f.irc.SendMessages(e.ReplyTarget(), content)
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
		Jar: f.ctx.RedditCookieJar(),
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
	req.Header.Set("User-Agent", f.cfg.Reddit.UserAgent)

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

	f.ctx.SetRedditModhash(body.JSON.Data.Modhash)
	f.ctx.SetRedditJWT(body.JSON.Data.Cookie)
	u, err := url.Parse("https://reddit.com")
	if err != nil {
		return err
	}

	f.ctx.RedditCookieJar().SetCookies(u, resp.Cookies())
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
