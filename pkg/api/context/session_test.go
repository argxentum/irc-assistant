package context

import (
	"fmt"
	"sync"
	"testing"
)

func TestSessionState(t *testing.T) {
	session := NewSession()

	if session.StartedAt().IsZero() {
		t.Fatal("session start time is zero")
	}
	if !session.IsAwake() {
		t.Fatal("new session is asleep")
	}

	if !session.SetAwake(false) {
		t.Fatal("sleep transition was not reported as a change")
	}
	if session.IsAwake() {
		t.Fatal("session remained awake")
	}
	if session.SetAwake(false) {
		t.Fatal("repeated sleep transition was reported as a change")
	}

	reddit := RedditSession{AccessToken: "token", ExpiresIn: 60}
	session.SetReddit(reddit)
	if got := session.Reddit(); got != reddit {
		t.Fatalf("Reddit() = %#v, want %#v", got, reddit)
	}

	session.AddBannedWord("#channel", "word")
	words := session.BannedWords("#channel")
	if !words["word"] {
		t.Fatal("added banned word is missing")
	}

	// BannedWords returns a snapshot so callers cannot mutate session state
	// after the session lock has been released.
	words["outside"] = true
	if session.BannedWords("#channel")["outside"] {
		t.Fatal("mutating a banned-word snapshot changed session state")
	}

	session.RemoveBannedWord("#channel", "word")
	if session.BannedWords("#channel")["word"] {
		t.Fatal("removed banned word remains in session state")
	}
}

func TestSessionConcurrentAccess(t *testing.T) {
	session := NewSession()
	const workers = 100

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			word := fmt.Sprintf("word-%d", i)
			session.AddBannedWord("#channel", word)
			_ = session.BannedWords("#channel")
			session.SetAwake(i%2 == 0)
			_ = session.IsAwake()
			session.SetReddit(RedditSession{AccessToken: word, ExpiresIn: 60})
			_ = session.Reddit()
			session.RemoveBannedWord("#channel", word)
		}()
	}
	wg.Wait()
}
