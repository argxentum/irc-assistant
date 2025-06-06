package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
	"os"
)

func main() {
	panic("Remove panic in main to load again...")

	ctx := context.NewContext()

	configFilename := ""
	channel := ""

	if len(os.Args) > 1 {
		configFilename = os.Args[1]
	}
	if len(os.Args) > 2 {
		channel = os.Args[2]
	}

	if len(configFilename) == 0 {
		panic("config filename is required")
	}

	if len(channel) == 0 {
		panic("channel is required")
	}

	cfg, err := config.ReadConfig(configFilename)
	if err != nil {
		panic(err)
	}

	initializeFirestore(ctx, cfg)
	defer firestore.Get().Close()

	initializeLogger(ctx, cfg)
	defer log.Logger().Close()

	if err = createDisinformationSources(channel); err != nil {
		panic(fmt.Errorf("error copying disinfo, %s", err))
	}
}

func initializeFirestore(ctx context.Context, cfg *config.Config) {
	_, err := firestore.Initialize(ctx, cfg)
	if err != nil {
		panic(fmt.Errorf("error initializing firestore, %s", err))
	}
}

func initializeLogger(ctx context.Context, cfg *config.Config) {
	if _, err := log.InitializeGCPLogger(ctx, cfg, "antifa-source-loader"); err != nil {
		panic(fmt.Errorf("error initializing GCP logger, %s", err))
	}
}

func createDisinformationSources(channel string) error {
	fs := firestore.Get()
	logger := log.Logger()

	ch, err := fs.Channel(channel)
	if err != nil {
		return fmt.Errorf("error getting channel %s, %v", channel, err)
	}

	for _, dw := range ch.Summarization.DisinformationWarnings {
		if err := fs.AddDisinformationSource(channel, dw); err != nil {
			logger.Warningf(nil, "error adding disinformation source %s for channel %s, %v", dw, channel, err)
		}
	}

	return nil
}
