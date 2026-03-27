package models

import "time"

type ProxySummaryRequestTaskData struct {
	RequestID string `json:"request_id,omitempty"`
	Channel   string `json:"channel"`
	Nick      string `json:"nick"`
	URL       string `json:"url"`
}

// NewProxySummaryRequestTask creates a fire-and-forget proxy summary request.
// The proxy will send the summary directly to IRC without the caller waiting.
func NewProxySummaryRequestTask(channel, nick, url string) *Task {
	return newTask(TaskTypeProxySummaryRequest, time.Now(), ProxySummaryRequestTaskData{
		Channel: channel,
		Nick:    nick,
		URL:     url,
	})
}

// NewProxySummaryRequestTaskWithWaiter creates a proxy summary request that the
// caller will wait on. The requestID correlates the response back to the waiting goroutine.
func NewProxySummaryRequestTaskWithWaiter(requestID, channel, nick, url string) *Task {
	return newTask(TaskTypeProxySummaryRequest, time.Now(), ProxySummaryRequestTaskData{
		RequestID: requestID,
		Channel:   channel,
		Nick:      nick,
		URL:       url,
	})
}

type ProxySummaryResponseTaskData struct {
	RequestID string   `json:"request_id,omitempty"`
	Channel   string   `json:"channel"`
	Nick      string   `json:"nick"`
	URL       string   `json:"url,omitempty"`
	Messages  []string `json:"messages"`
}

// NewProxySummaryResponseTask creates a response for a fire-and-forget request.
// The response is sent directly to IRC by the task processor.
func NewProxySummaryResponseTask(channel, nick, url string, messages []string) *Task {
	return newTask(TaskTypeProxySummaryResponse, time.Now(), ProxySummaryResponseTaskData{
		Channel:  channel,
		Nick:     nick,
		URL:      url,
		Messages: messages,
	})
}

// NewProxySummaryResponseTaskWithWaiter creates a response for a waiting request.
// The response is routed to the waiting goroutine via the requestID.
func NewProxySummaryResponseTaskWithWaiter(requestID, channel, nick, url string, messages []string) *Task {
	return newTask(TaskTypeProxySummaryResponse, time.Now(), ProxySummaryResponseTaskData{
		RequestID: requestID,
		Channel:   channel,
		Nick:      nick,
		URL:       url,
		Messages:  messages,
	})
}
