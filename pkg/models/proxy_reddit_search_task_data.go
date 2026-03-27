package models

import "time"

const (
	RedditSearchSortNew       = "new"
	RedditSearchSortRelevance = "relevance"
)

type ProxyRedditSearchRequestTaskData struct {
	Channel   string `json:"channel"`
	Nick      string `json:"nick"`
	Subreddit string `json:"subreddit"`
	Query     string `json:"query"`
	Sort      string `json:"sort"`
}

func NewProxyRedditSearchRequestTask(channel, nick, subreddit, query, sort string) *Task {
	return newTask(TaskTypeProxyRedditSearchRequest, time.Now(), ProxyRedditSearchRequestTaskData{
		Channel:   channel,
		Nick:      nick,
		Subreddit: subreddit,
		Query:     query,
		Sort:      sort,
	})
}

type ProxyRedditSearchResponseTaskData struct {
	Channel   string `json:"channel"`
	Nick      string `json:"nick"`
	Subreddit string `json:"subreddit"`
	Query     string `json:"query"`
	Posts     []any  `json:"posts"`
}

func NewProxyRedditSearchResponseTask(channel, nick, subreddit, query string, posts []any) *Task {
	return newTask(TaskTypeProxyRedditSearchResponse, time.Now(), ProxyRedditSearchResponseTaskData{
		Channel:   channel,
		Nick:      nick,
		Subreddit: subreddit,
		Query:     query,
		Posts:     posts,
	})
}
