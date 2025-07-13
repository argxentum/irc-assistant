package context

import (
	"time"
)

type Session struct {
	StartedAt   time.Time
	IsAwake     bool
	Reddit      RedditSession
	bannedWords []ChannelBannedWords
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
		bannedWords: make([]ChannelBannedWords, 0),
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

type ChannelBannedWords struct {
	channel string
	words   map[string]bool
}

func (s *Session) IsBannedWord(channel, word string) bool {
	for _, bw := range s.bannedWords {
		if bw.channel == channel {
			_, ok := bw.words[word]
			return ok
		}
	}

	return false
}

func (s *Session) AddBannedWord(channel, word string) {
	found := false
	for i, bw := range s.bannedWords {
		if bw.channel == channel {
			found = true
			s.bannedWords[i].words[word] = true
		}
	}

	if !found {
		s.bannedWords = append(s.bannedWords, ChannelBannedWords{
			channel: channel,
			words:   map[string]bool{word: true},
		})
	}
}

func (s *Session) RemoveBannedWord(channel, word string) {
	for i, bw := range s.bannedWords {
		empty := false

		if bw.channel == channel {
			delete(s.bannedWords[i].words, word)
			if len(s.bannedWords[i].words) == 0 {
				empty = true
			}
		}

		if empty {
			s.bannedWords = append(s.bannedWords[:i], s.bannedWords[i+1:]...)
		}
	}
}
