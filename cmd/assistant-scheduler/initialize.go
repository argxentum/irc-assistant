package main

import (
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/queue"
	"context"
	"fmt"
)

func initializeLogger(ctx context.Context, cfg *config.Config) {
	_, err := log.InitializeGCPLogger(ctx, cfg, fmt.Sprintf("%s-scheduler", cfg.IRC.Nick))
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

func initializeQueue(ctx context.Context, cfg *config.Config) {
	_, err := queue.Initialize(ctx, cfg)
	if err != nil {
		panic(fmt.Errorf("error initializing queue, %s", err))
	}
}
