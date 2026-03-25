package models

import "time"

type ProxySummaryRequestTaskData struct {
	Channel string `json:"channel"`
	Nick    string `json:"nick"`
	URL     string `json:"url"`
}

func NewProxySummaryRequestTask(channel, nick, url string) *Task {
	return newTask(TaskTypeProxySummaryRequest, time.Now(), ProxySummaryRequestTaskData{
		Channel: channel,
		Nick:    nick,
		URL:     url,
	})
}

type ProxySummaryResponseTaskData struct {
	Channel  string   `json:"channel"`
	Nick     string   `json:"nick"`
	URL      string   `json:"url"`
	Messages []string `json:"messages"`
}

func NewProxySummaryResponseTask(channel, nick, url string, messages []string) *Task {
	return newTask(TaskTypeProxySummaryResponse, time.Now(), ProxySummaryResponseTaskData{
		Channel:  channel,
		Nick:     nick,
		URL:      url,
		Messages: messages,
	})
}
