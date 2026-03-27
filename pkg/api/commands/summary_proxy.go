package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"sync"
	"time"

	"github.com/google/uuid"
)

const proxySummaryTimeout = 10 * time.Second

type proxySummaryResponse struct {
	title    string
	messages []string
}

// proxySummaryWaiters tracks pending proxy summary requests.
// When a proxy summary response arrives with a matching RequestID,
// the result is sent on the channel so the waiting summarization
// goroutine can use it.
var (
	proxySummaryMu      sync.Mutex
	proxySummaryWaiters = make(map[string]chan proxySummaryResponse)
)

// RegisterProxySummaryWaiter registers a channel to receive the proxy
// summary response for the given request ID.
func RegisterProxySummaryWaiter(requestID string) chan proxySummaryResponse {
	ch := make(chan proxySummaryResponse, 1)
	proxySummaryMu.Lock()
	proxySummaryWaiters[requestID] = ch
	proxySummaryMu.Unlock()
	return ch
}

// ResolveProxySummaryWaiter sends the result to a waiting summarization
// goroutine if one exists. Returns true if a waiter was found.
func ResolveProxySummaryWaiter(requestID, title string, messages []string) bool {
	proxySummaryMu.Lock()
	ch, ok := proxySummaryWaiters[requestID]
	if ok {
		delete(proxySummaryWaiters, requestID)
	}
	proxySummaryMu.Unlock()

	if ok {
		ch <- proxySummaryResponse{title: title, messages: messages}
		return true
	}
	return false
}

func removeProxySummaryWaiter(requestID string) {
	proxySummaryMu.Lock()
	delete(proxySummaryWaiters, requestID)
	proxySummaryMu.Unlock()
}

func (c *SummaryCommand) proxySummaryRequest(e *irc.Event, doc *retriever.Document) (*summaryResult, error) {
	logger := log.Logger()
	logger.Debugf(e, "attempting proxy summary for %s", doc.URL)

	requestID := uuid.NewString()
	ch := RegisterProxySummaryWaiter(requestID)
	defer removeProxySummaryWaiter(requestID)

	task := models.NewProxySummaryRequestTaskWithWaiter(requestID, e.ReplyTarget(), e.From, doc.URL)
	if err := queue.GetProxy().Publish(task); err != nil {
		logger.Errorf(e, "error publishing proxy summary request: %s", err)
		return nil, err
	}

	select {
	case resp := <-ch:
		if len(resp.messages) == 0 {
			logger.Debugf(e, "proxy summary returned empty for %s", doc.URL)
			return nil, nil
		}
		if c.isRejectedTitle(resp.title) {
			logger.Debugf(e, "rejected proxy summary title: %s", resp.title)
			return nil, nil
		}
		logger.Debugf(e, "proxy summary received for %s: %d messages", doc.URL, len(resp.messages))
		return createSummaryResult(resp.messages...), nil
	case <-time.After(proxySummaryTimeout):
		logger.Debugf(e, "proxy summary timed out for %s", doc.URL)
		return nil, nil
	}
}
