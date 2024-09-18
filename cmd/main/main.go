package main

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/handler"
)

func main() {
	ctx := context.NewContext()

	cfg, err := config.ReadConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	irc := core.NewIRC(ctx)
	err = irc.Connect(cfg, nil)
	if err != nil {
		panic(err)
	}

	ech := make(chan *core.Event)
	go irc.Listen(ech)

	h := handler.NewEventHandler(ctx, cfg, irc)
	for {
		e := <-ech
		go h.Handle(e)
	}
}
