package main

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
	"strings"
)

func processTasks(irc irc.IRC) {
	logger := log.Logger()

	go func() {
		err := queue.Get().Receive(func(task *models.Task) {
			logger.Debugf(nil, "received task %s: %s", task.ID, task.Type)

			switch task.Type {
			case models.TaskTypeReminder:
				processReminder(irc, task)
			case models.TaskTypeBanRemoval:
				processBanRemoval(irc, task)
			}
		})

		if err != nil {
			logger.Errorf(nil, "error processing due tasks, %s", err)
		}
	}()
}

func processReminder(irc irc.IRC, task *models.Task) {
	data := task.Data.(models.ReminderTaskData)

	logger := log.Logger()
	logger.Debugf(nil, "processing reminder for %s: %s", data.User, data.Content)

	message := ""
	if strings.HasPrefix(data.Destination, "#") || strings.HasPrefix(data.Destination, "&") {
		message = fmt.Sprintf("%s: here's the reminder you set %s: %s", data.User, elapse.TimeDescription(task.CreatedAt), style.Bold(data.Content))
	} else {
		message = fmt.Sprintf("Here's the reminder you set %s: %s", elapse.TimeDescription(task.CreatedAt), style.Bold(data.Content))
	}

	irc.SendMessage(data.Destination, message)
}

func processBanRemoval(irc irc.IRC, task *models.Task) {
	data := task.Data.(models.BanRemovalTaskData)

	logger := log.Logger()
	logger.Debugf(nil, "processing ban removal for %s in %s", data.Mask, data.Channel)

	irc.Unban(data.Channel, data.Mask)
}
