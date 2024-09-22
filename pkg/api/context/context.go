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
}

func NewContext() Context {
	jar, _ := cookiejar.New(nil)

	return &assistantContext{
		ctx: context.Background(),
		properties: map[string]any{
			startedAtKey:       time.Now(),
			isAwakeKey:         true,
			redditCookieJarKey: jar,
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
	c.Lock()
	defer c.Unlock()
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
