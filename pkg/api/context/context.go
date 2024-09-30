package context

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"
)

const (
	startedAtKey       = "startedAt"
	isAwakeKey         = "isAwake"
	redditJWTKey       = "redditJWT"
	redditModhashKey   = "redditModhash"
	redditCookieJarKey = "redditCookieJar"
	bannedWordsKey     = "bannedWords"
)

type Context interface {
	context.Context
	StartedAt() time.Time
	SetStartedAt(time.Time)
	IsAwake() bool
	SetAwake(bool)
	RedditJWT() string
	SetRedditJWT(string)
	RedditModhash() string
	SetRedditModhash(string)
	RedditCookieJar() http.CookieJar
	BannedWords(string) map[string]bool
	SetBannedWords(string, []string)
	AddBannedWord(string, string)
	RemoveBannedWord(string, string)
}

func NewContext() Context {
	jar, _ := cookiejar.New(nil)

	return &assistantContext{
		ctx: context.Background(),
		properties: map[string]any{
			startedAtKey:       time.Now(),
			isAwakeKey:         true,
			redditCookieJarKey: jar,
			bannedWordsKey:     make(map[string]map[string]bool),
		},
	}
}

type assistantContext struct {
	context.Context
	sync.Mutex
	ctx        context.Context
	properties map[string]any
}

func (c *assistantContext) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *assistantContext) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *assistantContext) Err() error {
	return c.ctx.Err()
}

func (c *assistantContext) Value(k any) any {
	return c.properties[fmt.Sprintf("%s", k)]
}

func (c *assistantContext) set(k any, v any) {
	c.Lock()
	defer c.Unlock()
	c.properties[fmt.Sprintf("%s", k)] = v
}

func (c *assistantContext) StartedAt() time.Time {
	return c.Value(startedAtKey).(time.Time)
}

func (c *assistantContext) SetStartedAt(t time.Time) {
	c.set(startedAtKey, t)
}

func (c *assistantContext) IsAwake() bool {
	return c.Value(isAwakeKey).(bool)
}

func (c *assistantContext) SetAwake(awake bool) {
	c.set(isAwakeKey, awake)
}

func (c *assistantContext) RedditJWT() string {
	if c.properties[redditJWTKey] == nil {
		return ""
	}
	return c.Value(redditJWTKey).(string)
}

func (c *assistantContext) SetRedditJWT(jwt string) {
	c.set(redditJWTKey, jwt)
}

func (c *assistantContext) RedditModhash() string {
	if c.properties[redditModhashKey] == nil {
		return ""
	}
	return c.Value(redditModhashKey).(string)
}

func (c *assistantContext) SetRedditModhash(modhash string) {
	c.set(redditModhashKey, modhash)
}

func (c *assistantContext) RedditCookieJar() http.CookieJar {
	return c.Value(redditCookieJarKey).(http.CookieJar)
}

func (c *assistantContext) BannedWords(channel string) map[string]bool {
	bannedWords := c.Value(bannedWordsKey).(map[string]map[string]bool)
	return bannedWords[channel]
}

func (c *assistantContext) SetBannedWords(channel string, words []string) {
	bannedWords := c.Value(bannedWordsKey).(map[string]map[string]bool)

	bannedWords[channel] = make(map[string]bool)

	for _, word := range words {
		bannedWords[channel][word] = true
	}

	c.set(bannedWordsKey, bannedWords)
}

func (c *assistantContext) AddBannedWord(channel, word string) {
	bannedWords := c.Value(bannedWordsKey).(map[string]map[string]bool)
	if bannedWords[channel] == nil {
		bannedWords[channel] = make(map[string]bool)
	}
	bannedWords[channel][word] = true
	c.set(bannedWordsKey, bannedWords)
}

func (c *assistantContext) RemoveBannedWord(channel, word string) {
	bannedWords := c.Value(bannedWordsKey).(map[string]map[string]bool)
	delete(bannedWords[channel], word)
	c.set(bannedWordsKey, bannedWords)
}
