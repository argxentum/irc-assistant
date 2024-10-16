package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type redditFunction struct {
	*functionStub
	subreddit   string
	description string
	triggers    []string
	usages      []string
	retriever   retriever.DocumentRetriever
}

func NewRedditFunction(ctx context.Context, cfg *config.Config, irc irc.IRC, subreddit, description string, triggers, usages []string) Function {
	return &redditFunction{
		functionStub: defaultFunctionStub(ctx, cfg, irc),
		subreddit:    subreddit,
		description:  description,
		triggers:     triggers,
		usages:       usages,
		retriever:    retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (f *redditFunction) Name() string {
	return fmt.Sprintf("r/%s", f.subreddit)
}

func (f *redditFunction) Description() string {
	return f.description
}

func (f *redditFunction) Triggers() []string {
	return f.triggers
}

func (f *redditFunction) Usages() []string {
	return f.usages
}

func (f *redditFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *redditFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *redditFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	query := strings.Join(tokens[1:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), query)

	if reddit.IsJWTExpired(f.ctx.Session().Reddit.JWT) {
		logger.Debug(e, "reddit JWT token expired, logging in")
		result, err := reddit.Login(f.cfg.Reddit.Username, f.cfg.Reddit.Password)
		if err != nil {
			logger.Errorf(e, "error logging into reddit: %s", err)
			return
		}

		if result == nil {
			logger.Errorf(e, "unable to login to reddit")
			return
		}

		f.ctx.Session().Reddit.JWT = result.JWT
		f.ctx.Session().Reddit.Modhash = result.Modhash
		f.ctx.Session().Reddit.CookieJar.SetCookies(result.URL, result.Cookies)
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

func (f *redditFunction) sendPostMessages(e *irc.Event, posts []reddit.Post) {
	content := make([]string, 0)
	for i, post := range posts {
		title := post.Title
		if len(title) == 0 {
			continue
		}
		if len(title) > postsTitleMaxLength {
			title = title[:postsTitleMaxLength] + "..."
		}
		content = append(content, fmt.Sprintf("%s (r/%s, %s)", style.Bold(title), f.subreddit, elapse.TimeDescription(time.Unix(int64(post.Created), 0))))
		content = append(content, post.URL)
		if i < len(posts)-1 {
			content = append(content, " ")
		}
	}

	f.SendMessages(e, e.ReplyTarget(), content)
}

const topRedditPosts = "https://api.reddit.com/r/%s/top.json?limit=%d"
const searchRedditPosts = "https://api.reddit.com/r/%s/search.json?sort=new&limit=1&restrict_sr=on&q=title:%s"
const defaultRedditPosts = 3
const maxRedditPosts = 5

func topSubredditPosts(subreddit string, n int) ([]reddit.Post, error) {
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

	var listing reddit.Listing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	posts := make([]reddit.Post, 0)
	for _, child := range listing.Data.Children {
		posts = append(posts, child.Data)
	}

	return posts, nil
}

func (f *redditFunction) searchNewSubredditPosts(topic string) ([]reddit.Post, error) {
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

	defer resp.Body.Close()

	var listing reddit.Listing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	posts := make([]reddit.Post, 0)
	for _, child := range listing.Data.Children {
		posts = append(posts, child.Data)
	}

	return posts, nil
}
