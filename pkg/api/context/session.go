package context

import (
	"sync"
	"time"
)

type Session struct {
	mu          sync.RWMutex
	startedAt   time.Time
	isAwake     bool
	reddit      RedditSession
	bannedWords map[string]map[string]bool // channel -> word/phrase -> true
}

type RedditSession struct {
	AccessToken string  `json:"access_token"`
	ExpiresIn   float64 `json:"expires_in"`
}

func (rs RedditSession) IsExpired() bool {
	if len(rs.AccessToken) == 0 || rs.ExpiresIn <= 0 {
		return true
	}

	expirationTime := time.Now().Add(time.Duration(rs.ExpiresIn) * time.Second)
	return time.Now().After(expirationTime)
}

func NewSession() *Session {
	return &Session{
		startedAt:   time.Now(),
		isAwake:     true,
		reddit:      RedditSession{},
		bannedWords: make(map[string]map[string]bool),
	}
}

func (s *Session) StartedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startedAt
}

func (s *Session) IsAwake() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isAwake
}

func (s *Session) SetAwake(awake bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := s.isAwake != awake
	s.isAwake = awake
	return changed
}

func (s *Session) Reddit() RedditSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reddit
}

func (s *Session) SetReddit(reddit RedditSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reddit = reddit
}

type Cache struct {
	properties map[string]any
}

func (c *Cache) Get(k string) any {
	return c.properties[k]
}

func (c *Cache) Set(k string, v any) {
	c.properties[k] = v
}

func (s *Session) BannedWords(channel string) map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	words := s.bannedWords[channel]
	result := make(map[string]bool, len(words))
	for word, banned := range words {
		result[word] = banned
	}
	return result
}

func (s *Session) AddBannedWord(channel, word string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bannedWords[channel] == nil {
		s.bannedWords[channel] = make(map[string]bool)
	}
	s.bannedWords[channel][word] = true
}

func (s *Session) RemoveBannedWord(channel, word string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if words, ok := s.bannedWords[channel]; ok {
		delete(words, word)
		if len(words) == 0 {
			delete(s.bannedWords, channel)
		}
	}
}
