package stats

import "sync"

var (
	mu       sync.Mutex
	messages = make(map[string]int)
)

func IncrementMessages(channel string) {
	mu.Lock()
	messages[channel]++
	mu.Unlock()
}

func ReadAndResetMessages(channel string) int {
	mu.Lock()
	count := messages[channel]
	messages[channel] = 0
	mu.Unlock()
	return count
}
