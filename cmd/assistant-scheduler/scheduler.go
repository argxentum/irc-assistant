package main

import (
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"context"
	"fmt"
	"time"
)

type scheduler struct {
	ctx context.Context
	cfg *config.Config
}

func (s *scheduler) start() {
	logger := log.Logger()
	fs := firestore.Get()
	logger.Debug(nil, "starting scheduler")

	err := queue.Get().Clear()
	if err != nil {
		panic(fmt.Errorf("error clearing queue, %s", err))
	}

	for {
		tasks := make([]*models.Task, 0)

		scheduled, err := fs.DueTasks()
		if err != nil {
			logger.Errorf(nil, "error getting due tasks, %s", err)
		}
		tasks = append(tasks, scheduled...)

		persistent, err := fs.PersistentTasks(models.ChannelInactivityTaskID)
		if err != nil {
			logger.Errorf(nil, "error getting persistent tasks, %s", err)
		}
		tasks = append(tasks, persistent...)

		publishTasks(tasks)
		time.Sleep(15 * time.Second)
	}
}

func publishTasks(tasks []*models.Task) {
	if len(tasks) == 0 {
		return
	}

	logger := log.Logger()
	q := queue.Get()

	for _, task := range tasks {
		if task == nil {
			logger.Warningf(nil, "skipping nil task")
			continue
		}

		logger.Debugf(nil, "publishing %s: %s", task.ID, task.Type)

		if err := q.Publish(task); err != nil {
			logger.Errorf(nil, "error publishing %s, %s", task.ID, err)
			continue
		}
	}
}
