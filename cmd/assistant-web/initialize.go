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
	_, err := log.InitializeGCPLogger(ctx, cfg, cfg.Web.Domain)
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
	_, err := queue.InitializeDefault(ctx, cfg, cfg.Queue.Topic, "")
	if err != nil {
		panic(fmt.Errorf("error initializing queue, %s", err))
	}
	_, err = queue.InitializeDashboard(ctx, cfg, cfg.Web.Dashboard.Queue.Topic, cfg.Web.Dashboard.Queue.Subscription)
	if err != nil {
		panic(fmt.Errorf("error initializing dashboard queue, %s", err))
	}
}
