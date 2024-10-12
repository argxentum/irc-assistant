package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"os"
)

const defaultConfigFilename = "config.yaml"

func main() {
	serviceCtx := context.NewContext()

	configFilename := defaultConfigFilename
	if len(os.Args) > 1 {
		configFilename = os.Args[1]
	}

	cfg, err := config.ReadConfig(configFilename)
	if err != nil {
		panic(err)
	}

	log.InitializeGCPLogger(serviceCtx, cfg)
	defer log.Logger().Close()

	s := &server{
		cfg: cfg,
	}

	s.start()
}
