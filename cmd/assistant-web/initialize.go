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
	_, err = queue.InitializeDashboardRequest(ctx, cfg, cfg.Web.Dashboard.RequestQueue.Topic, "")
	if err != nil {
		panic(fmt.Errorf("error initializing dashboard request queue, %s", err))
	}
	_, err = queue.InitializeDashboardResponse(ctx, cfg, cfg.Web.Dashboard.ResponseQueue.Topic, cfg.Web.Dashboard.ResponseQueue.Subscription)
	if err != nil {
		panic(fmt.Errorf("error initializing dashboard response queue, %s", err))
	}
}
