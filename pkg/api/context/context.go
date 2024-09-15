package context

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	StartedAtKey = "startedAt"
)

type Context interface {
	context.Context
	Set(k any, v any)
}

func NewContext() Context {
	return &assistantContext{
		ctx:        context.Background(),
		properties: map[string]any{StartedAtKey: time.Now()},
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

func (c *assistantContext) Set(k any, v any) {
	c.Lock()
	defer c.Unlock()
	c.properties[fmt.Sprintf("%s", k)] = v
}
