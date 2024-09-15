package handler

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/functions"
	"fmt"
	"slices"
)

type EventHandler interface {
	ReloadFunctions()
	AddFunction(f functions.Function)
	FindMatchingFunction(e *core.Event) functions.Function
	Handle(e *core.Event)
}

type eventHandler struct {
	ctx context.Context
	cfg *config.Config
	irc core.IRC
	fn  []functions.Function
}

func NewEventHandler(ctx context.Context, cfg *config.Config, irc core.IRC) EventHandler {
	eh := &eventHandler{
		ctx: ctx,
		cfg: cfg,
		irc: irc,
		fn:  make([]functions.Function, 0),
	}

	eh.ReloadFunctions()
	return eh
}

func (eh *eventHandler) ReloadFunctions() {
	eh.fn = make([]functions.Function, 0)
	for name := range eh.cfg.Functions.EnabledFunctions {
		f, err := functions.NewFunction(eh.ctx, eh.cfg, eh.irc, name)
		if err != nil {
			fmt.Printf("error loading function: %s\n", name)
			continue
		}
		eh.fn = append(eh.fn, f)
	}
}

func (eh *eventHandler) AddFunction(f functions.Function) {
	eh.fn = append(eh.fn, f)
}

func (eh *eventHandler) FindMatchingFunction(e *core.Event) functions.Function {
	for _, f := range eh.fn {
		if f.ShouldExecute(e) {
			return f
		}
	}
	return nil
}

func (eh *eventHandler) Handle(e *core.Event) {
	core.LogEvent(e)

	switch e.Code {
	case core.CodeInvite:
		// if the sender of invite is the owner or an admin, join the channel
		sender, _ := e.Sender()
		if sender == eh.cfg.Connection.Owner || slices.Contains(eh.cfg.Connection.Admins, sender) {
			channel := e.Arguments[1]
			eh.irc.Join(channel)
		}
	case core.CodePrivateMessage:
		if f := eh.FindMatchingFunction(e); f != nil {
			err := f.Execute(e)
			if err != nil {
				recipient, _ := e.Recipient()
				eh.irc.SendMessage(recipient, err.Error())
			}
		}
	}
}
