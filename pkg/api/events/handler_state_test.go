package events

import (
	"assistant/pkg/api/irc"
	"sync"
	"testing"
	"time"
)

func newStateOnlyHandler() *handler {
	return &handler{
		messageHistory:              make(map[string]time.Time),
		rateLimitCounter:            make(map[string]int),
		temporarilyIgnoredUserMasks: make(map[string]int64),
	}
}

func TestHandlerRateLimitStateBelongsToHandler(t *testing.T) {
	event := &irc.Event{Source: "nick!user@host"}
	first := newStateOnlyHandler()
	second := newStateOnlyHandler()

	first.updateUserCommandHistory(event)
	if !first.isUserCommandRateLimited(event) {
		t.Fatal("first handler did not retain its message history")
	}
	if second.isUserCommandRateLimited(event) {
		t.Fatal("message history leaked into another handler")
	}
}

func TestHandlerConcurrentRateLimitAccess(t *testing.T) {
	event := &irc.Event{Source: "nick!user@host"}
	handler := newStateOnlyHandler()
	handler.updateUserCommandHistory(event)

	const workers = 100
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !handler.isUserCommandRateLimited(event) {
				t.Error("recent command was not rate limited")
			}
			handler.updateUserCommandHistory(event)
		}()
	}
	wg.Wait()
}
