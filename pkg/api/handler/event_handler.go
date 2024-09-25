package handler

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/functions"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"slices"
	"strings"
)

type EventHandler interface {
	ReloadFunctions()
	AddFunction(f functions.Function)
	FindMatchingFunction(e *irc.Event) functions.Function
	Handle(e *irc.Event)
}

type eventHandler struct {
	ctx context.Context
	cfg *config.Config
	irc irc.IRC
	fn  []functions.Function
}

func NewEventHandler(ctx context.Context, cfg *config.Config, irc irc.IRC) EventHandler {
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
		f, err := functions.Route(eh.ctx, eh.cfg, eh.irc, name)
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

func (eh *eventHandler) FindMatchingFunction(e *irc.Event) functions.Function {
	for _, f := range eh.fn {
		if f.MayExecute(e) {
			return f
		}
	}
	return nil
}

func (eh *eventHandler) Handle(e *irc.Event) {
	logger := log.Logger()
	logger.Default(e, e.Raw)

	switch e.Code {
	case irc.CodeInvite:
		// if the sender of invite is the owner or an admin, join the channel
		sender, _ := e.Sender()
		if sender == eh.cfg.Connection.Owner || slices.Contains(eh.cfg.Connection.Admins, sender) {
			channel := e.Arguments[1]
			eh.irc.Join(channel)
		}
	case irc.CodePrivateMessage:
		if f := eh.FindMatchingFunction(e); f != nil {
			f.IsAuthorized(e, func(authorized bool) {
				tokens := functions.Tokens(e.Message())

				if !authorized {
					logger.Warningf(e, "unauthorized attempt by %s to use %s", e.From, tokens[0])

					if strings.HasPrefix(tokens[0], eh.cfg.Functions.Prefix) {
						f.Replyf(e, "You are not authorized to use %s.", style.Bold(strings.TrimPrefix(tokens[0], eh.cfg.Functions.Prefix)))
					} else {
						f.Replyf(e, "You are not authorized to perform that command.")
					}
					return
				}

				go f.Execute(e)
			})
		}
	}
}
