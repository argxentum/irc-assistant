package models

import (
	"time"
)

type ProxyInactivityRequestTaskData struct {
	Channel   string `json:"channel"`
	Subreddit string `json:"subreddit"`
	Category  string `json:"category"`
	Count     int    `json:"count"`
}

func NewProxyInactivityRequestTask(channel, subreddit, category string, count int) *Task {
	return newTask(TaskTypeProxyInactivityRequest, time.Now(), ProxyInactivityRequestTaskData{
		Channel:   channel,
		Subreddit: subreddit,
		Category:  category,
		Count:     count,
	})
}

type ProxyInactivityResponseTaskData struct {
	Channel string `json:"channel"`
	Posts   []any  `json:"posts"`
}

func NewProxyInactivityResponseTask(channel string, posts []any) *Task {
	return newTask(TaskTypeProxyInactivityResponse, time.Now(), ProxyInactivityResponseTaskData{
		Channel: channel,
		Posts:   posts,
	})
}
