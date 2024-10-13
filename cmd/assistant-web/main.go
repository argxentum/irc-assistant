package main

import (
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"context"
	"os"
)

const defaultConfigFilename = "config.yaml"

func main() {
	serviceCtx := context.Background()

	configFilename := defaultConfigFilename
	if len(os.Args) > 1 {
		configFilename = os.Args[1]
	}

	cfg, err := config.ReadConfig(configFilename)
	if err != nil {
		panic(err)
	}

	initializeLogger(serviceCtx, cfg)
	defer log.Logger().Close()

	initializeFirestore(serviceCtx, cfg)
	defer firestore.Get().Close()

	s := &server{
		cfg: cfg,
	}

	s.start()
}
