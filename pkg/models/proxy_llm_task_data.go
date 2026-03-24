package models

import "time"

type ProxyLLMRequestTaskData struct {
	Handler string `firestore:"handler" json:"handler"`
	Channel string `firestore:"channel" json:"channel"`
	Nick    string `firestore:"nick" json:"nick"`
	Prompt  string `firestore:"prompt" json:"prompt"`
}

func NewProxyLLMRequestTask(channel, nick, handler, prompt string) *Task {
	return newTask(TaskTypeProxyLLMRequest, time.Now(), ProxyLLMRequestTaskData{
		Handler: handler,
		Channel: channel,
		Nick:    nick,
		Prompt:  prompt,
	})
}

type ProxyLLMResponseTaskData struct {
	RequestID string   `firestore:"request_id" json:"request_id"`
	Channel   string   `firestore:"channel" json:"channel"`
	Nick      string   `firestore:"nick" json:"nick"`
	Messages  []string `firestore:"messages" json:"messages"`
}

func NewProxyLLMResponseTask(requestID, channel, nick string, messages []string) *Task {
	return newTask(TaskTypeProxyLLMResponse, time.Now(), ProxyLLMResponseTaskData{
		RequestID: requestID,
		Channel:   channel,
		Nick:      nick,
		Messages:  messages,
	})
}
