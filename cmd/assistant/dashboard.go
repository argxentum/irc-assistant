package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
)

func processDashboardRequests(ctx context.Context, cfg *config.Config, ircs irc.IRC) {
	logger := log.Logger()

	go func() {
		err := queue.GetDashboard().Receive(func(task *models.Task) {
			if task.Type != models.TaskTypeDashboardRequest {
				return
			}

			data := task.Data.(models.DashboardRequestTaskData)
			logger.Debugf(nil, "dashboard request: %s [%s]", data.Action, data.Channel)

			var resp *models.Task

			switch data.Action {
			case models.DashboardActionListUsers:
				resp = handleDashboardListUsers(ircs, data)
			default:
				resp = models.NewDashboardResponseTask(data.RequestID, data.Action, false, "unknown action", nil)
			}

			if err := queue.GetDashboard().Publish(resp); err != nil {
				logger.Errorf(nil, "error publishing dashboard response: %s", err)
			}
		})

		if err != nil {
			logger.Errorf(nil, "error receiving dashboard requests: %s", err)
		}
	}()
}

func handleDashboardListUsers(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	done := make(chan []*irc.User, 1)
	ircs.ListUsers(data.Channel, func(users []*irc.User) {
		done <- users
	})
	users := <-done

	dashUsers := make([]models.DashboardUser, 0, len(users))
	for _, u := range users {
		dashUsers = append(dashUsers, models.DashboardUser{
			Nick:   u.Mask.Nick,
			User:   u.Mask.UserID,
			Host:   u.Mask.Host,
			Status: string(u.Status),
		})
	}

	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", dashUsers)
}
