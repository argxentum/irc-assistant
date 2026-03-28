package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
	"time"
)

func processDashboardRequests(ctx context.Context, cfg *config.Config, ircs irc.IRC) {
	logger := log.Logger()

	go func() {
		err := queue.GetDashboardRequest().Receive(func(task *models.Task) {
			if task.Type != models.TaskTypeDashboardRequest {
				return
			}

			data := task.Data.(models.DashboardRequestTaskData)
			logger.Debugf(nil, "dashboard request: %s [%s]", data.Action, data.Channel)

			var resp *models.Task

			switch data.Action {
			case models.DashboardActionListUsers:
				resp = handleDashboardListUsers(ircs, data)
			case models.DashboardActionKick:
				resp = handleDashboardUserAction(ircs, data)
			case models.DashboardActionBan:
				resp = handleDashboardUserAction(ircs, data)
			case models.DashboardActionMute:
				resp = handleDashboardUserAction(ircs, data)
			case models.DashboardActionUnban:
				resp = handleDashboardUserAction(ircs, data)
			case models.DashboardActionUnmute:
				resp = handleDashboardUserAction(ircs, data)
			default:
				resp = models.NewDashboardResponseTask(data.RequestID, data.Action, false, "unknown action", nil)
			}

			if err := queue.GetDashboardResponse().Publish(resp); err != nil {
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
	ircs.ListUsersByMask(data.Channel, "*!*@*", func(users []*irc.User) {
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

func handleDashboardUserAction(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	logger := log.Logger()

	// look up the target user to verify they exist and check their status
	done := make(chan *irc.User, 1)
	ircs.GetUser(data.Channel, data.Nick, func(user *irc.User) {
		done <- user
	})
	user := <-done

	if user == nil {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "user not found", nil)
	}

	// only allow actions on voiced and normal users
	if user.Status == irc.ChannelStatusOperator || user.Status == irc.ChannelStatusHalfOperator {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false,
			fmt.Sprintf("%s is %s and cannot be targeted", data.Nick, irc.StatusName(user.Status)), nil)
	}

	reason := data.Reason
	if reason == "" {
		reason = "dashboard action"
	}

	switch data.Action {
	case models.DashboardActionKick:
		ircs.Kick(data.Channel, data.Nick, reason)
		logger.Infof(nil, "dashboard: kicked %s from %s", data.Nick, data.Channel)

	case models.DashboardActionBan:
		mask := fmt.Sprintf("*!*@%s", user.Mask.Host)
		if data.Duration != "" {
			dur, err := elapse.ParseDuration(data.Duration)
			if err == nil {
				reason = fmt.Sprintf("%s - temporarily banned for %s", reason, elapse.ParseDurationDescription(data.Duration))
				task := models.NewBanRemovalTask(time.Now().Add(dur), mask, data.Channel)
				if err := firestore.Get().AddTask(task); err != nil {
					logger.Errorf(nil, "dashboard: error scheduling ban removal: %s", err)
				}
			}
		}
		ircs.Kick(data.Channel, data.Nick, reason)
		ircs.Ban(data.Channel, mask)
		logger.Infof(nil, "dashboard: banned %s (%s) from %s", data.Nick, mask, data.Channel)

	case models.DashboardActionMute:
		ircs.Mute(data.Channel, data.Nick)
		if data.Duration != "" {
			dur, err := elapse.ParseDuration(data.Duration)
			if err == nil {
				task := models.NewMuteRemovalTask(time.Now().Add(dur), data.Channel, data.Nick, user.Mask.Host, false)
				if err := firestore.Get().AddTask(task); err != nil {
					logger.Errorf(nil, "dashboard: error scheduling mute removal: %s", err)
				}
			}
		}
		logger.Infof(nil, "dashboard: muted %s in %s", data.Nick, data.Channel)

	case models.DashboardActionUnban:
		mask := fmt.Sprintf("*!*@%s", user.Mask.Host)
		ircs.Unban(data.Channel, mask)
		logger.Infof(nil, "dashboard: unbanned %s (%s) from %s", data.Nick, mask, data.Channel)

	case models.DashboardActionUnmute:
		ircs.Voice(data.Channel, data.Nick)
		logger.Infof(nil, "dashboard: unmuted %s in %s", data.Nick, data.Channel)
	}

	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", nil)
}
