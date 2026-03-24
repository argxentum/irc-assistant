package models

import "time"

type ProxyLLMRequestTaskData struct {
	Handler string `json:"handler"`
	Channel string `json:"channel"`
	Nick    string `json:"nick"`
	Prompt  string `json:"prompt"`
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
	RequestID  string `json:"request_id"`
	Channel    string `json:"channel"`
	Nick       string `json:"nick"`
	ResponseID string `json:"response_id"`
	SessionID  string `json:"session_id"`
}

func NewProxyLLMResponseTask(requestID, channel, nick, responseID, sessionID string) *Task {
	return newTask(TaskTypeProxyLLMResponse, time.Now(), ProxyLLMResponseTaskData{
		RequestID:  requestID,
		Channel:    channel,
		Nick:       nick,
		ResponseID: responseID,
		SessionID:  sessionID,
	})
}
