package context

import (
	"time"
)

type Session struct {
	StartedAt   time.Time
	IsAwake     bool
	Reddit      RedditSession
	bannedWords map[string]map[string]bool // channel -> word/phrase -> true
}

type RedditSession struct {
	AccessToken string  `json:"access_token"`
	ExpiresIn   float64 `json:"expires_in"`
}

func (rs *RedditSession) IsExpired() bool {
	if len(rs.AccessToken) == 0 || rs.ExpiresIn <= 0 {
		return true
	}

	expirationTime := time.Now().Add(time.Duration(rs.ExpiresIn) * time.Second)
	return time.Now().After(expirationTime)
}

func NewSession() *Session {
	return &Session{
		StartedAt:   time.Now(),
		IsAwake:     true,
		Reddit:      RedditSession{},
		bannedWords: make(map[string]map[string]bool),
	}
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
	return s.bannedWords[channel]
}

func (s *Session) AddBannedWord(channel, word string) {
	if s.bannedWords[channel] == nil {
		s.bannedWords[channel] = make(map[string]bool)
	}
	s.bannedWords[channel][word] = true
}

func (s *Session) RemoveBannedWord(channel, word string) {
	if words, ok := s.bannedWords[channel]; ok {
		delete(words, word)
		if len(words) == 0 {
			delete(s.bannedWords, channel)
		}
	}
}
