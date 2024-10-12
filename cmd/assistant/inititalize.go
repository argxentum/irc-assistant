package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
)

func initializeLogger(ctx context.Context, cfg *config.Config) {
	_, err := log.InitializeGCPLogger(ctx, cfg, cfg.IRC.Nick)
	if err != nil {
		panic(fmt.Errorf("error initializing logger, %s", err))
	}
}

func initializeFirestore(ctx context.Context, cfg *config.Config) {
	_, err := firestore.Initialize(ctx, cfg)
	if err != nil {
		panic(fmt.Errorf("error initializing firestore, %s", err))
	}
}

func initializeChannel(ctx context.Context, channel string) {
	logger := log.Logger()
	logger.Rawf(log.Debug, "loading banned words for channel %s", channel)

	bannedWords, err := firestore.Get().BannedWords(ctx, channel)
	if err != nil {
		panic(fmt.Errorf("error retrieving banned words, %s", err))
	}

	for _, word := range bannedWords {
		ctx.Session().AddBannedWord(channel, word.Word)
	}

	logger.Rawf(log.Debug, "loaded %d banned words for channel %s", len(bannedWords), channel)
}
