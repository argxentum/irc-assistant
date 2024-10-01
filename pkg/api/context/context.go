package context

import (
	"context"
	"sync"
	"time"
)

type Context interface {
	context.Context
	Session() *Session
}

func NewContext() Context {
	return &assistantContext{
		ctx:     context.Background(),
		session: NewSession(),
	}
}

type assistantContext struct {
	context.Context
	sync.Mutex
	ctx     context.Context
	session *Session
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
	return nil
}

func (c *assistantContext) Session() *Session {
	c.Lock()
	defer c.Unlock()
	return c.session
}
