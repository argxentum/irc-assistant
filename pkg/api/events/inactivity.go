package events

import (
	"sync"
	"time"
)

const maxInactivityPersistenceInterval = 30 * time.Second

type stoppableTimer interface {
	Stop() bool
}

type inactivityChannelState struct {
	persistMu     sync.Mutex
	lastActivity  time.Time
	lastPersisted time.Time
	duration      time.Duration
	timer         stoppableTimer
	generation    uint64
}

type inactivityTracker struct {
	mu        sync.Mutex
	channels  map[string]*inactivityChannelState
	now       func() time.Time
	afterFunc func(time.Duration, func()) stoppableTimer
	persist   func(string, time.Time) error
	onError   func(string, error)
}

func newInactivityTracker(persist func(string, time.Time) error, onError func(string, error)) *inactivityTracker {
	return &inactivityTracker{
		channels: make(map[string]*inactivityChannelState),
		now:      time.Now,
		afterFunc: func(delay time.Duration, callback func()) stoppableTimer {
			return time.AfterFunc(delay, callback)
		},
		persist: persist,
		onError: onError,
	}
}

func (t *inactivityTracker) RecordActivity(channel string, duration time.Duration) error {
	now := t.now()
	interval := inactivityPersistenceInterval(duration)

	t.mu.Lock()
	state := t.channels[channel]
	if state == nil {
		state = &inactivityChannelState{}
		t.channels[channel] = state
	}
	if state.timer != nil {
		state.timer.Stop()
		state.timer = nil
	}

	state.lastActivity = now
	state.duration = duration
	state.generation++
	generation := state.generation

	if state.lastPersisted.IsZero() || now.Sub(state.lastPersisted) >= interval {
		t.mu.Unlock()

		persisted, err := t.persistIfCurrent(channel, generation, now.Add(duration))
		if err == nil && persisted {
			t.markPersistenceSucceeded(channel, generation, now)
		}
		return err
	}

	state.timer = t.afterFunc(interval, func() {
		if err := t.flush(channel, generation); err != nil && t.onError != nil {
			t.onError(channel, err)
		}
	})
	t.mu.Unlock()
	return nil
}

func (t *inactivityTracker) flush(channel string, generation uint64) error {
	t.mu.Lock()
	state := t.channels[channel]
	if state == nil || state.generation != generation {
		t.mu.Unlock()
		return nil
	}

	activityAt := state.lastActivity
	dueAt := activityAt.Add(state.duration)
	state.timer = nil
	t.mu.Unlock()

	persisted, err := t.persistIfCurrent(channel, generation, dueAt)
	if err == nil && persisted {
		t.markPersistenceSucceeded(channel, generation, activityAt)
	}
	return err
}

func (t *inactivityTracker) persistIfCurrent(channel string, generation uint64, dueAt time.Time) (bool, error) {
	t.mu.Lock()
	state := t.channels[channel]
	t.mu.Unlock()
	if state == nil {
		return false, nil
	}

	// Serialize writes for a channel, then recheck the generation so a delayed
	// timer can never overwrite a newer due time.
	state.persistMu.Lock()
	defer state.persistMu.Unlock()

	t.mu.Lock()
	current := state.generation == generation
	t.mu.Unlock()
	if !current {
		return false, nil
	}

	return true, t.persist(channel, dueAt)
}

func (t *inactivityTracker) markPersistenceSucceeded(channel string, generation uint64, activityAt time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state := t.channels[channel]
	if state != nil && state.generation == generation {
		state.lastPersisted = activityAt
	}
}

func inactivityPersistenceInterval(duration time.Duration) time.Duration {
	interval := duration / 2
	if interval > maxInactivityPersistenceInterval {
		return maxInactivityPersistenceInterval
	}
	if interval <= 0 {
		return time.Nanosecond
	}
	return interval
}
