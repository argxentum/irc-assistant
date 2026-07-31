package irc

import (
	"strings"
	"sync"
	"time"

	irce "github.com/thoj/go-ircevent"
)

const ircRequestTimeout = 5 * time.Second

type callbackRegistry interface {
	AddCallback(eventCode string, callback func(*irce.Event)) int
	RemoveCallback(eventCode string, id int) bool
}

// ircRequestManager keeps request/response callbacks bounded and correlated.
// Requests run in goroutines so neither the caller nor the IRC read loop waits
// for a later numeric response.
type ircRequestManager struct {
	registry callbackRegistry
	send     func(string)
	timeout  time.Duration
	gates    sync.Map
}

func newIRCRequestManager(registry callbackRegistry, send func(string), timeout time.Duration) *ircRequestManager {
	return &ircRequestManager{
		registry: registry,
		send:     send,
		timeout:  timeout,
	}
}

// run registers handlers before sending command. A handler returns true when
// its event completes the request. Requests with the same key are serialized;
// requests for different targets can proceed concurrently.
func (m *ircRequestManager) run(
	key string,
	command string,
	handlers map[string]func(*irce.Event) bool,
	complete func(timedOut bool),
) {
	go func() {
		deadline := time.Now().Add(m.timeout)
		release, ok := m.acquire(key, deadline)
		if !ok {
			go complete(true)
			return
		}

		var stateMu sync.Mutex
		finished := false
		callbackIDs := make(map[string]int, len(handlers))
		var timer *time.Timer

		cleanup := func() {
			if timer != nil {
				timer.Stop()
			}
			for code, id := range callbackIDs {
				m.registry.RemoveCallback(code, id)
			}
			release()
		}

		finish := func(timedOut bool) {
			stateMu.Lock()
			if finished {
				stateMu.Unlock()
				return
			}
			finished = true
			stateMu.Unlock()

			cleanup()
			// The consumer callback may start another IRC request or perform
			// synchronous work, so never run it on go-ircevent's callback path.
			go complete(timedOut)
		}

		// Keep callbacks from observing a partially populated callbackIDs map.
		stateMu.Lock()
		for code, handler := range handlers {
			code := code
			handler := handler
			callbackIDs[code] = m.registry.AddCallback(code, func(event *irce.Event) {
				stateMu.Lock()
				if finished {
					stateMu.Unlock()
					return
				}
				done := handler(event)
				stateMu.Unlock()
				if done {
					finish(false)
				}
			})
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			stateMu.Unlock()
			finish(true)
			return
		}
		timer = time.AfterFunc(remaining, func() {
			finish(true)
		})
		stateMu.Unlock()

		// SendRaw can wait for the connection's writer. This entire operation
		// is already on a worker goroutine, while the deadline above remains
		// able to complete the request for its caller.
		m.send(command)
	}()
}

func (m *ircRequestManager) acquire(key string, deadline time.Time) (func(), bool) {
	gateValue, _ := m.gates.LoadOrStore(key, make(chan struct{}, 1))
	gate := gateValue.(chan struct{})

	wait := time.Until(deadline)
	if wait <= 0 {
		return nil, false
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case gate <- struct{}{}:
		return func() { <-gate }, true
	case <-timer.C:
		return nil, false
	}
}

func eventArgumentEquals(event *irce.Event, index int, expected string) bool {
	return len(event.Arguments) > index && strings.EqualFold(event.Arguments[index], expected)
}

func requestKey(command, target string) string {
	return strings.ToUpper(command) + ":" + strings.ToLower(target)
}
