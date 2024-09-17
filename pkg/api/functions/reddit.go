package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const redditFunctionPrefix = "reddit"

type redditFunction struct {
	Stub
	subreddit string
}

func NewRedditFunction(subreddit string, ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, fmt.Sprintf("%s/%s", redditFunctionPrefix, subreddit))
	if err != nil {
		return nil, err
	}

	return &redditFunction{
		Stub:      stub,
		subreddit: subreddit,
	}, nil
}

func (f *redditFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 1)
}

func (f *redditFunction) Execute(e *core.Event) {
	fmt.Printf("Executing function: reddit/%s\n", f.subreddit)
	tokens := Tokens(e.Message())
	query := strings.Join(tokens[1:], " ")
	posts, err := SearchNewPosts(f.subreddit, query)
	if err != nil {
		f.Reply(e, "Unable to retrieve r/%s posts", f.subreddit)
		return
	}
	if len(posts) == 0 {
		f.Reply(e, "No r/%s posts found for %s", f.subreddit, text.Bold(query))
		return
	}
	f.showResults(e, posts)
}

func (f *redditFunction) showResults(e *core.Event, posts []RedditPost) {
	content := make([]string, 0)
	for i, post := range posts {
		title := post.Title
		if len(title) > 100 {
			title = title[:100] + "..."
		} else if len(title) == 0 {
			title = "No title"
		}
		content = append(content, text.Bold(title))
		content = append(content, time.Unix(int64(post.Created), 0).Format("January 2, 2006"))
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

func TopPosts(subreddit string, n int) ([]RedditPost, error) {
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

func SearchNewPosts(subreddit, topic string) ([]RedditPost, error) {
	t := url.QueryEscape(topic)
	query := fmt.Sprintf(searchRedditPosts, subreddit, t)

	// create request
	req, err := http.NewRequest("GET", query, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	resp, err := http.DefaultClient.Do(req)
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
