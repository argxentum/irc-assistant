package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/events"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/queue"
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

	initializeLogger(ctx, cfg)
	defer log.Logger().Close()

	initializeFirestore(ctx, cfg)
	defer firestore.Get().Close()

	initializeQueue(ctx, cfg)
	defer queue.Get().Close()

	svc := irc.NewIRC(ctx)
	err = svc.Connect(cfg, initializeAssistant, func(channel string, mask *irc.Mask) {
		if mask.Nick == cfg.IRC.Nick {
			initializeChannel(ctx, cfg, svc, channel)
		} else {
			initializeChannelUser(ctx, cfg, svc, channel, mask)
		}
	})
	if err != nil {
		panic(err)
	}

	ech := make(chan *irc.Event)
	go svc.Listen(ech)

	h := events.NewHandler(ctx, cfg, svc)
	for {
		e := <-ech
		h.Handle(e)
	}
}
