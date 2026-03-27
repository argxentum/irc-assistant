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

// proxySummaryWaiters tracks pending proxy summary requests.
// When a proxy summary response arrives with a matching RequestID,
// the result is sent on the channel so the waiting summarization
// goroutine can use it.
var (
	proxySummaryMu      sync.Mutex
	proxySummaryWaiters = make(map[string]chan []string)
)

// RegisterProxySummaryWaiter registers a channel to receive the proxy
// summary response for the given request ID.
func RegisterProxySummaryWaiter(requestID string) chan []string {
	ch := make(chan []string, 1)
	proxySummaryMu.Lock()
	proxySummaryWaiters[requestID] = ch
	proxySummaryMu.Unlock()
	return ch
}

// ResolveProxySummaryWaiter sends the result to a waiting summarization
// goroutine if one exists. Returns true if a waiter was found.
func ResolveProxySummaryWaiter(requestID string, messages []string) bool {
	proxySummaryMu.Lock()
	ch, ok := proxySummaryWaiters[requestID]
	if ok {
		delete(proxySummaryWaiters, requestID)
	}
	proxySummaryMu.Unlock()

	if ok {
		ch <- messages
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
	case messages := <-ch:
		if len(messages) == 0 {
			logger.Debugf(e, "proxy summary returned empty for %s", doc.URL)
			return nil, nil
		}
		logger.Debugf(e, "proxy summary received for %s: %d messages", doc.URL, len(messages))
		return createSummaryResult(messages...), nil
	case <-time.After(proxySummaryTimeout):
		logger.Debugf(e, "proxy summary timed out for %s", doc.URL)
		return nil, nil
	}
}
