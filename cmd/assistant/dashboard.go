package main

import (
	"assistant/pkg/api/actions"
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
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
			case models.DashboardActionAddBan:
				resp = handleDashboardAddBan(ircs, data)
			case models.DashboardActionListBans:
				resp = handleDashboardListBans(ircs, data)
			case models.DashboardActionExpireBan:
				resp = handleDashboardExpireBan(ircs, data)
			case models.DashboardActionExpireMute:
				resp = handleDashboardExpireMute(ircs, data)
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

	switch data.Action {
	case models.DashboardActionKick:
		ircs.Kick(data.Channel, data.Nick, data.Reason)
		logger.Infof(nil, "dashboard: kicked %s from %s", data.Nick, data.Channel)

	case models.DashboardActionBan:
		mask := fmt.Sprintf("*!*@%s", user.Mask.Host)
		actions.Ban(ircs, data.Channel, mask, data.Duration, data.Reason)

	case models.DashboardActionMute:
		actions.Mute(ircs, data.Channel, data.Nick, user.Mask.Host, data.Duration, data.Reason)

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

func handleDashboardAddBan(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	logger := log.Logger()

	if data.Mask == "" {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "mask is required", nil)
	}

	ircs.Ban(data.Channel, data.Mask)
	logger.Infof(nil, "dashboard: added ban %s in %s", data.Mask, data.Channel)

	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", nil)
}

func handleDashboardExpireBan(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	logger := log.Logger()

	if data.Mask == "" {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "mask is required", nil)
	}

	ircs.Unban(data.Channel, data.Mask)
	logger.Infof(nil, "dashboard: expired ban %s from %s", data.Mask, data.Channel)

	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", nil)
}

func handleDashboardExpireMute(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	logger := log.Logger()

	if data.Nick == "" {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "nick is required", nil)
	}

	ircs.Voice(data.Channel, data.Nick)
	logger.Infof(nil, "dashboard: expired mute for %s in %s", data.Nick, data.Channel)

	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", nil)
}

func handleDashboardListBans(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	done := make(chan []*irc.BanEntry, 1)
	ircs.ListBans(data.Channel, func(bans []*irc.BanEntry) {
		done <- bans
	})
	bans := <-done

	dashBans := make([]models.DashboardBan, 0, len(bans))
	for _, b := range bans {
		entry := models.DashboardBan{
			Mask:  b.Mask,
			SetBy: b.SetBy,
		}
		if b.SetAt != nil {
			entry.SetAt = b.SetAt.Unix()
		}
		dashBans = append(dashBans, entry)
	}

	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", dashBans)
}
