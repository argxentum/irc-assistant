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
	_, err := log.InitializeGCPLogger(ctx, cfg, fmt.Sprintf("%s-proxy", cfg.IRC.Nick))
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

func initializeQueues(ctx context.Context, cfg *config.Config) {
	_, err := queue.InitializeDefault(ctx, cfg, cfg.Queue.Topic, cfg.Queue.Subscription)
	if err != nil {
		panic(fmt.Errorf("error default queue, %s", err))
	}

	_, err = queue.InitializeProxy(ctx, cfg, cfg.Proxy.Queue.Topic, cfg.Proxy.Queue.Subscription)
	if err != nil {
		panic(fmt.Errorf("error initializing proxy queue, %s", err))
	}
}
