package events

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeInactivityTimer struct {
	mu       sync.Mutex
	stopped  bool
	callback func()
}

func (t *fakeInactivityTimer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	wasActive := !t.stopped
	t.stopped = true
	return wasActive
}

func (t *fakeInactivityTimer) Fire() {
	t.callback()
}

type persistedInactivity struct {
	channel string
	dueAt   time.Time
}

func TestInactivityTrackerBoundsPersistenceDuringMessageBurst(t *testing.T) {
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	duration := 2 * time.Minute
	var persisted []persistedInactivity
	var timers []*fakeInactivityTimer

	tracker := newInactivityTracker(func(channel string, dueAt time.Time) error {
		persisted = append(persisted, persistedInactivity{channel: channel, dueAt: dueAt})
		return nil
	}, nil)
	tracker.now = func() time.Time { return now }
	tracker.afterFunc = func(_ time.Duration, callback func()) stoppableTimer {
		timer := &fakeInactivityTimer{callback: callback}
		timers = append(timers, timer)
		return timer
	}

	if err := tracker.RecordActivity("#channel", duration); err != nil {
		t.Fatalf("record first activity: %v", err)
	}
	if len(persisted) != 1 || persisted[0].dueAt != now.Add(duration) {
		t.Fatalf("first persistence = %#v, want due at %s", persisted, now.Add(duration))
	}

	now = now.Add(10 * time.Second)
	if err := tracker.RecordActivity("#channel", duration); err != nil {
		t.Fatalf("record second activity: %v", err)
	}
	if len(persisted) != 1 || len(timers) != 1 {
		t.Fatalf("second activity persisted %d times and created %d timers", len(persisted), len(timers))
	}

	now = now.Add(10 * time.Second)
	if err := tracker.RecordActivity("#channel", duration); err != nil {
		t.Fatalf("record third activity: %v", err)
	}
	if len(persisted) != 1 || len(timers) != 2 {
		t.Fatalf("third activity persisted %d times and created %d timers", len(persisted), len(timers))
	}

	// A stopped timer may already be racing with Stop. Its generation must keep
	// it from persisting stale activity.
	timers[0].Fire()
	if len(persisted) != 1 {
		t.Fatalf("stale timer persisted activity: %#v", persisted)
	}

	timers[1].Fire()
	if len(persisted) != 2 {
		t.Fatalf("trailing activity persisted %d times, want 2", len(persisted))
	}
	if want := now.Add(duration); persisted[1].dueAt != want {
		t.Fatalf("trailing due time = %s, want %s", persisted[1].dueAt, want)
	}

	now = now.Add(31 * time.Second)
	if err := tracker.RecordActivity("#channel", duration); err != nil {
		t.Fatalf("record activity after interval: %v", err)
	}
	if len(persisted) != 3 {
		t.Fatalf("activity after interval persisted %d times, want 3", len(persisted))
	}
}

func TestInactivityTrackerKeepsChannelsIndependent(t *testing.T) {
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	var persisted []persistedInactivity
	tracker := newInactivityTracker(func(channel string, dueAt time.Time) error {
		persisted = append(persisted, persistedInactivity{channel: channel, dueAt: dueAt})
		return nil
	}, nil)
	tracker.now = func() time.Time { return now }

	if err := tracker.RecordActivity("#one", time.Minute); err != nil {
		t.Fatalf("record #one: %v", err)
	}
	if err := tracker.RecordActivity("#two", 2*time.Minute); err != nil {
		t.Fatalf("record #two: %v", err)
	}

	if len(persisted) != 2 || persisted[0].channel != "#one" || persisted[1].channel != "#two" {
		t.Fatalf("independent channel persistence = %#v", persisted)
	}
}

func TestInactivityTrackerRetriesAfterPersistenceFailure(t *testing.T) {
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	attempts := 0
	tracker := newInactivityTracker(func(string, time.Time) error {
		attempts++
		if attempts == 1 {
			return errors.New("temporary failure")
		}
		return nil
	}, nil)
	tracker.now = func() time.Time { return now }

	if err := tracker.RecordActivity("#channel", time.Minute); err == nil {
		t.Fatal("first persistence succeeded")
	}
	now = now.Add(time.Second)
	if err := tracker.RecordActivity("#channel", time.Minute); err != nil {
		t.Fatalf("retry persistence: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("persistence attempts = %d, want 2", attempts)
	}
}

func TestInactivityTrackerSerializesConcurrentPersistence(t *testing.T) {
	currentTime := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	duration := 2 * time.Minute
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var mu sync.Mutex
	var persisted []time.Time

	tracker := newInactivityTracker(func(_ string, dueAt time.Time) error {
		mu.Lock()
		first := len(persisted) == 0
		if first {
			persisted = append(persisted, dueAt)
		}
		mu.Unlock()

		if first {
			close(firstStarted)
			<-releaseFirst
			return nil
		}

		mu.Lock()
		persisted = append(persisted, dueAt)
		mu.Unlock()
		return nil
	}, nil)
	tracker.now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return currentTime
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tracker.RecordActivity("#channel", duration); err != nil {
			t.Errorf("record first activity: %v", err)
		}
	}()
	<-firstStarted

	mu.Lock()
	currentTime = currentTime.Add(31 * time.Second)
	latestActivity := currentTime
	mu.Unlock()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tracker.RecordActivity("#channel", duration); err != nil {
			t.Errorf("record concurrent activity: %v", err)
		}
	}()

	close(releaseFirst)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(persisted) != 2 {
		t.Fatalf("persisted %d due times, want 2", len(persisted))
	}
	if want := latestActivity.Add(duration); persisted[1] != want {
		t.Fatalf("latest persisted due time = %s, want %s", persisted[1], want)
	}
}

func TestInactivityPersistenceInterval(t *testing.T) {
	if got := inactivityPersistenceInterval(2 * time.Minute); got != maxInactivityPersistenceInterval {
		t.Fatalf("long interval = %s, want %s", got, maxInactivityPersistenceInterval)
	}
	if got := inactivityPersistenceInterval(10 * time.Second); got != 5*time.Second {
		t.Fatalf("short interval = %s, want 5s", got)
	}
}
