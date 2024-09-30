package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
)

func initializeLogger(ctx context.Context, cfg *config.Config) {
	_, err := log.InitializeGCPLogger(ctx, cfg)
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

func initializeBannedWords(ctx context.Context, cfg *config.Config) {
	bannedWords, err := firestore.Get().AllBannedWords(ctx)
	if err != nil {
		panic(fmt.Errorf("error retrieving banned words, %s", err))
	}

	byChannel := make(map[string]map[string]bool)

	for _, bw := range bannedWords {
		if bw.Bot != cfg.Connection.Nick || bw.Server != cfg.Connection.ServerName {
			continue
		}
		if byChannel[bw.Channel] == nil {
			byChannel[bw.Channel] = make(map[string]bool)
		}
		byChannel[bw.Channel][bw.Word] = true
	}

	for channel, words := range byChannel {
		bw := make([]string, 0)
		for word := range words {
			bw = append(bw, word)
		}
		ctx.SetBannedWords(channel, bw)
	}
}
