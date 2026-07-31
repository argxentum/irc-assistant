package irc

import (
	"sync"
	"testing"
	"time"

	irce "github.com/thoj/go-ircevent"
)

type fakeCallbackRegistry struct {
	mu        sync.Mutex
	nextID    int
	callbacks map[string]map[int]func(*irce.Event)
}

func newFakeCallbackRegistry() *fakeCallbackRegistry {
	return &fakeCallbackRegistry{callbacks: make(map[string]map[int]func(*irce.Event))}
}

func (r *fakeCallbackRegistry) AddCallback(code string, callback func(*irce.Event)) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.callbacks[code] == nil {
		r.callbacks[code] = make(map[int]func(*irce.Event))
	}
	id := r.nextID
	r.nextID++
	r.callbacks[code][id] = callback
	return id
}

func (r *fakeCallbackRegistry) RemoveCallback(code string, id int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.callbacks[code] == nil {
		return false
	}
	delete(r.callbacks[code], id)
	return true
}

func (r *fakeCallbackRegistry) emit(event *irce.Event) {
	r.mu.Lock()
	callbacks := make([]func(*irce.Event), 0, len(r.callbacks[event.Code]))
	for _, callback := range r.callbacks[event.Code] {
		callbacks = append(callbacks, callback)
	}
	r.mu.Unlock()

	for _, callback := range callbacks {
		callback(event)
	}
}

func (r *fakeCallbackRegistry) callbackCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, callbacks := range r.callbacks {
		count += len(callbacks)
	}
	return count
}

func newTestService(registry *fakeCallbackRegistry, send func(string), timeout time.Duration) *service {
	return &service{requests: newIRCRequestManager(registry, send, timeout)}
}

func receiveCommand(t *testing.T, commands <-chan string) string {
	t.Helper()
	select {
	case command := <-commands:
		return command
	case <-time.After(time.Second):
		t.Fatal("IRC command was not sent")
		return ""
	}
}

func receiveUsers(t *testing.T, users <-chan []*User) []*User {
	t.Helper()
	select {
	case result := <-users:
		return result
	case <-time.After(time.Second):
		t.Fatal("IRC user callback was not called")
		return nil
	}
}

func TestListUsersCorrelatesConcurrentChannelReplies(t *testing.T) {
	registry := newFakeCallbackRegistry()
	commands := make(chan string, 2)
	s := newTestService(registry, func(command string) { commands <- command }, time.Second)
	oneResult := make(chan []*User, 1)
	twoResult := make(chan []*User, 1)

	s.ListUsers("#one", func(users []*User) { oneResult <- users })
	s.ListUsers("#two", func(users []*User) { twoResult <- users })
	receiveCommand(t, commands)
	receiveCommand(t, commands)

	registry.emit(&irce.Event{Code: CodeNamesReply, Arguments: []string{"assistant", "=", "#two", "+bob"}})
	registry.emit(&irce.Event{Code: CodeNamesReply, Arguments: []string{"assistant", "=", "#one", "@alice"}})
	registry.emit(&irce.Event{Code: CodeEndOfNames, Arguments: []string{"assistant", "#one", "End of NAMES"}})
	registry.emit(&irce.Event{Code: CodeEndOfNames, Arguments: []string{"assistant", "#two", "End of NAMES"}})

	one := receiveUsers(t, oneResult)
	two := receiveUsers(t, twoResult)
	if len(one) != 1 || one[0].Mask.Nick != "alice" || one[0].Status != ChannelStatusOperator {
		t.Fatalf("#one users = %#v", one)
	}
	if len(two) != 1 || two[0].Mask.Nick != "bob" || two[0].Status != ChannelStatusVoice {
		t.Fatalf("#two users = %#v", two)
	}
}

func TestListUsersSerializesRequestsForSameChannel(t *testing.T) {
	registry := newFakeCallbackRegistry()
	commands := make(chan string, 2)
	s := newTestService(registry, func(command string) { commands <- command }, time.Second)
	firstResult := make(chan []*User, 1)
	secondResult := make(chan []*User, 1)

	s.ListUsers("#channel", func(users []*User) { firstResult <- users })
	if got := receiveCommand(t, commands); got != "NAMES #channel" {
		t.Fatalf("first command = %q", got)
	}

	s.ListUsers("#channel", func(users []*User) { secondResult <- users })
	select {
	case command := <-commands:
		t.Fatalf("duplicate request sent before first completed: %q", command)
	case <-time.After(30 * time.Millisecond):
	}

	registry.emit(&irce.Event{Code: CodeNamesReply, Arguments: []string{"assistant", "=", "#channel", "alice"}})
	registry.emit(&irce.Event{Code: CodeEndOfNames, Arguments: []string{"assistant", "#channel", "End of NAMES"}})
	first := receiveUsers(t, firstResult)
	if len(first) != 1 || first[0].Mask.Nick != "alice" {
		t.Fatalf("first users = %#v", first)
	}

	if got := receiveCommand(t, commands); got != "NAMES #channel" {
		t.Fatalf("second command = %q", got)
	}
	registry.emit(&irce.Event{Code: CodeNamesReply, Arguments: []string{"assistant", "=", "#channel", "bob"}})
	registry.emit(&irce.Event{Code: CodeEndOfNames, Arguments: []string{"assistant", "#channel", "End of NAMES"}})
	second := receiveUsers(t, secondResult)
	if len(second) != 1 || second[0].Mask.Nick != "bob" {
		t.Fatalf("second users = %#v", second)
	}
}

func TestIRCRequestHelperReturnsWhileSendIsBlocked(t *testing.T) {
	registry := newFakeCallbackRegistry()
	sendStarted := make(chan struct{})
	releaseSend := make(chan struct{})
	s := newTestService(registry, func(string) {
		close(sendStarted)
		<-releaseSend
	}, time.Second)
	result := make(chan string, 1)

	returned := make(chan struct{})
	go func() {
		s.GetTopic("#channel", func(topic string) { result <- topic })
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("GetTopic blocked its caller")
	}
	select {
	case <-sendStarted:
	case <-time.After(time.Second):
		t.Fatal("request worker did not try to send")
	}
	registry.emit(&irce.Event{Code: CodeTopicReply, Arguments: []string{"assistant", "#channel", "topic"}})
	select {
	case topic := <-result:
		if topic != "topic" {
			t.Fatalf("topic = %q", topic)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not call back")
	}

	if count := registry.callbackCount(); count != 0 {
		t.Fatalf("callbacks remaining after completion = %d", count)
	}
	close(releaseSend)
}

func TestIRCRequestTimesOutAndRemovesCallbacks(t *testing.T) {
	registry := newFakeCallbackRegistry()
	releaseSend := make(chan struct{})
	manager := newIRCRequestManager(registry, func(string) { <-releaseSend }, 40*time.Millisecond)
	result := make(chan bool, 1)

	manager.run("NAMES:#channel", "NAMES #channel", map[string]func(*irce.Event) bool{
		CodeEndOfNames: func(e *irce.Event) bool {
			return eventArgumentEquals(e, 1, "#channel")
		},
	}, func(timedOut bool) { result <- timedOut })

	select {
	case timedOut := <-result:
		if !timedOut {
			t.Fatal("request completed without timing out")
		}
	case <-time.After(time.Second):
		t.Fatal("timed-out request did not call back")
	}
	if count := registry.callbackCount(); count != 0 {
		t.Fatalf("callbacks remaining after timeout = %d", count)
	}
	close(releaseSend)
}

func TestIRCResponseDoesNotRunConsumerOnCallbackPath(t *testing.T) {
	registry := newFakeCallbackRegistry()
	commands := make(chan string, 1)
	s := newTestService(registry, func(command string) { commands <- command }, time.Second)
	consumerStarted := make(chan struct{})
	releaseConsumer := make(chan struct{})
	s.GetTopic("#channel", func(string) {
		close(consumerStarted)
		<-releaseConsumer
	})
	receiveCommand(t, commands)

	emitReturned := make(chan struct{})
	go func() {
		registry.emit(&irce.Event{Code: CodeTopicReply, Arguments: []string{"assistant", "#channel", "topic"}})
		close(emitReturned)
	}()
	select {
	case <-consumerStarted:
	case <-time.After(time.Second):
		t.Fatal("consumer callback did not start")
	}
	select {
	case <-emitReturned:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("consumer callback blocked the IRC callback path")
	}
	close(releaseConsumer)
}
