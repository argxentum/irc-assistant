package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/handler"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"os"
)

const defaultConfigFilename = "config.yaml"

func main() {
	ctx := context.NewContext()

	configFilename := defaultConfigFilename
	if len(os.Args) > 1 {
		configFilename = os.Args[1]
	}

	cfg, err := config.ReadConfig(configFilename)
	if err != nil {
		panic(err)
	}

	_, err = log.Initialize(ctx, cfg)
	if err != nil {
		panic(fmt.Errorf("error initializing logger, %s", err))
	}
	defer log.Logger().Close()

	svc := irc.NewIRC(ctx)
	err = svc.Connect(cfg, nil)
	if err != nil {
		panic(err)
	}

	ech := make(chan *irc.Event)
	go svc.Listen(ech)

	h := handler.NewEventHandler(ctx, cfg, svc)
	for {
		e := <-ech
		h.Handle(e)
	}
}
