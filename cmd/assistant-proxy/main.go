package main

import (
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/queue"
	"context"
	"os"
)

const defaultConfigFilename = "config.yaml"

func main() {
	ctx := context.Background()

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

	initializeQueues(ctx, cfg)
	defer queue.GetDefault().Close()
	defer queue.GetProxy().Close()

	p := &proxy{cfg: cfg, sessions: make(map[string]*session)}
	p.start()
}
