package commands

import (
	"sync"
	"testing"
	"time"
)

func TestSummaryPauseConcurrentCompletedUpdates(t *testing.T) {
	command := &SummaryCommand{userPauses: make(map[string]UserPause)}
	const updates = 100
	key := "nick@#channel"
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	for i := 0; i < updates; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			command.recordCompletedSummary(key, "#channel", "nick", now)
		}()
	}
	wg.Wait()

	pause, ok := command.getUserPause(key)
	if !ok {
		t.Fatal("pause was not recorded")
	}
	if pause.summaryCount != updates {
		t.Fatalf("summary count = %d, want %d", pause.summaryCount, updates)
	}
	if pause.channel != "#channel" || pause.nick != "nick" {
		t.Fatalf("pause identity = %s/%s, want #channel/nick", pause.channel, pause.nick)
	}
}

func TestSummaryPauseConcurrentIgnoredUpdates(t *testing.T) {
	command := &SummaryCommand{userPauses: make(map[string]UserPause)}
	const updates = 100
	key := "nick@#channel"
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	command.recordCompletedSummary(key, "#channel", "nick", now)

	var wg sync.WaitGroup
	for i := 0; i < updates; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, paused, _ := command.recordIgnoredSummaryIfPaused(key, now); !paused {
				t.Error("active pause was reported as expired")
			}
		}()
	}
	wg.Wait()

	pause, ok := command.getUserPause(key)
	if !ok {
		t.Fatal("pause was not recorded")
	}
	if pause.summaryCount != updates+1 {
		t.Fatalf("summary count = %d, want %d", pause.summaryCount, updates+1)
	}
	if pause.ignoreCount != updates {
		t.Fatalf("ignore count = %d, want %d", pause.ignoreCount, updates)
	}
}

func TestSummaryPauseExpiredResetAndSnapshot(t *testing.T) {
	command := &SummaryCommand{userPauses: make(map[string]UserPause)}
	key := "nick@#channel"
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	command.userPauses[key] = UserPause{
		channel:      "#channel",
		nick:         "nick",
		summaryCount: 5,
		ignoreCount:  3,
		timeoutAt:    now.Add(-time.Second),
	}

	pause, paused, existed := command.recordIgnoredSummaryIfPaused(key, now)
	if paused || !existed {
		t.Fatalf("expired pause result = paused %t, existed %t", paused, existed)
	}
	if pause.summaryCount != 0 || pause.ignoreCount != 0 || !pause.timeoutAt.IsZero() {
		t.Fatalf("expired pause was not reset: %#v", pause)
	}

	// getUserPause returns a value snapshot, not mutable shared state.
	pause.summaryCount = 99
	stored, _ := command.getUserPause(key)
	if stored.summaryCount != 0 {
		t.Fatalf("mutating pause snapshot changed stored count to %d", stored.summaryCount)
	}
}
