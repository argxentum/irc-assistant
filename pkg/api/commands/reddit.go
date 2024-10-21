package commands

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

type redditCommand struct {
	*commandStub
	subreddit   string
	description string
	triggers    []string
	usages      []string
	retriever   retriever.DocumentRetriever
}

func NewRedditCommand(ctx context.Context, cfg *config.Config, irc irc.IRC, subreddit, description string, triggers, usages []string) Command {
	return &redditCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		subreddit:   subreddit,
		description: description,
		triggers:    triggers,
		usages:      usages,
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *redditCommand) Name() string {
	return fmt.Sprintf("r/%s", c.subreddit)
}

func (c *redditCommand) Description() string {
	return c.description
}

func (c *redditCommand) Triggers() []string {
	return c.triggers
}

func (c *redditCommand) Usages() []string {
	return c.usages
}

func (c *redditCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *redditCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *redditCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	query := strings.Join(tokens[1:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), query)

	if reddit.IsJWTExpired(c.ctx.Session().Reddit.JWT) {
		logger.Debug(e, "reddit JWT token expired, logging in")
		result, err := reddit.Login(c.cfg.Reddit.Username, c.cfg.Reddit.Password)
		if err != nil {
			logger.Errorf(e, "error logging into reddit: %s", err)
			return
		}

		if result == nil {
			logger.Errorf(e, "unable to login to reddit")
			return
		}

		c.ctx.Session().Reddit.JWT = result.JWT
		c.ctx.Session().Reddit.Modhash = result.Modhash
		c.ctx.Session().Reddit.CookieJar.SetCookies(result.URL, result.Cookies)
	}

	posts, err := c.searchNewSubredditPosts(query)
	if err != nil {
		logger.Warningf(e, "unable to retrieve %s posts in r/%s: %s", query, c.subreddit, err)
		c.Replyf(e, "Unable to retrieve r/%s posts", c.subreddit)
		return
	}
	if len(posts) == 0 {
		logger.Warningf(e, "no %s posts in r/%s", query, c.subreddit)
		c.Replyf(e, "No r/%s posts found for %s", c.subreddit, style.Bold(query))
		return
	}
	c.sendPostMessages(e, posts)
}

const postsTitleMaxLength = 256

func (c *redditCommand) sendPostMessages(e *irc.Event, posts []reddit.Post) {
	content := make([]string, 0)
	for i, post := range posts {
		title := post.Title
		if len(title) == 0 {
			continue
		}
		if len(title) > postsTitleMaxLength {
			title = title[:postsTitleMaxLength] + "..."
		}
		content = append(content, fmt.Sprintf("%s (r/%s, %s)", style.Bold(title), c.subreddit, elapse.TimeDescription(time.Unix(int64(post.Created), 0))))
		content = append(content, post.URL)
		if i < len(posts)-1 {
			content = append(content, " ")
		}
	}

	c.SendMessages(e, e.ReplyTarget(), content)
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

func (c *redditCommand) searchNewSubredditPosts(topic string) ([]reddit.Post, error) {
	t := url.QueryEscape(topic)
	query := fmt.Sprintf(searchRedditPosts, c.subreddit, t)

	req, err := http.NewRequest(http.MethodGet, query, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.cfg.Reddit.UserAgent)

	client := &http.Client{
		Jar: c.ctx.Session().Reddit.CookieJar,
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
