package main

import (
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"context"
	"time"
)

type scheduler struct {
	ctx context.Context
	cfg *config.Config
}

func (s *scheduler) start() {
	logger := log.Logger()
	logger.Debug(nil, "starting scheduler")

	for {
		tasks, err := firestore.Get().DueTasks()

		if err != nil {
			logger.Errorf(nil, "error getting due tasks, %s", err)
		} else if len(tasks) > 0 {
			publishDueTasks(tasks)
		}

		time.Sleep(1 * time.Second)
	}
}

func publishDueTasks(tasks []*models.Task) {
	logger := log.Logger()
	q := queue.Get()
	fs := firestore.Get()

	for _, task := range tasks {
		logger.Debugf(nil, "publishing %s: %s", task.ID, task.Type)

		if err := q.Publish(task); err != nil {
			logger.Errorf(nil, "error publishing %s, %s", task.ID, err)
			continue
		}

		if err := fs.RemoveTask(task.ID); err != nil {
			logger.Errorf(nil, "error completing %s, %s", task.ID, err)
		}
	}
}
