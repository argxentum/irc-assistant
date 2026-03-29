package main

import (
	"assistant/pkg/api/actions"
	"assistant/pkg/api/commands"
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
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
			case models.DashboardActionGetTopic:
				resp = handleDashboardGetTopic(ircs, data)
			case models.DashboardActionSetTopic:
				resp = handleDashboardSetTopic(ircs, data)
			case models.DashboardActionApproveVR:
				resp = handleDashboardApproveVoiceRequest(ircs, data)
			case models.DashboardActionDenyVR:
				resp = handleDashboardDenyVoiceRequest(ircs, data)
			case models.DashboardActionListBans:
				resp = handleDashboardListBans(ircs, data)
			case models.DashboardActionExpireBan:
				resp = handleDashboardExpireBan(ircs, data)
			case models.DashboardActionExpireMute:
				resp = handleDashboardExpireMute(ircs, data)
			case models.DashboardActionListCommands:
				resp = handleDashboardListCommands(ctx, cfg, ircs, data)
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

func handleDashboardGetTopic(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	done := make(chan string, 1)
	ircs.GetTopic(data.Channel, func(topic string) {
		done <- topic
	})
	topic := <-done

	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", topic)
}

func handleDashboardSetTopic(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	logger := log.Logger()

	ircs.SetTopic(data.Channel, data.Topic)
	logger.Infof(nil, "dashboard: set topic in %s", data.Channel)

	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", nil)
}

func handleDashboardApproveVoiceRequest(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	logger := log.Logger()

	if data.Nick == "" {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "nick is required", nil)
	}

	fs := firestore.Get()
	ch, err := fs.Channel(data.Channel)
	if err != nil || ch == nil {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "channel not found", nil)
	}

	// find and remove the voice request
	found := false
	for _, vr := range ch.VoiceRequests {
		if vr.Nick == data.Nick {
			found = true
			break
		}
	}
	if !found {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "voice request not found", nil)
	}

	repository.RemoveChannelVoiceRequest(nil, ch, data.Nick, "")
	if err = repository.UpdateChannelVoiceRequests(nil, ch); err != nil {
		logger.Errorf(nil, "dashboard: error updating voice requests: %s", err)
	}

	// voice the user and set auto-voice
	ircs.Voice(data.Channel, data.Nick)

	u, err := repository.GetUserByNick(nil, data.Channel, data.Nick, true)
	if err != nil {
		logger.Errorf(nil, "dashboard: error getting user for auto-voice: %s", err)
	} else if u != nil {
		u.IsAutoVoiced = true
		if err = repository.UpdateUserIsAutoVoiced(nil, data.Channel, u); err != nil {
			logger.Errorf(nil, "dashboard: error updating auto-voice: %s", err)
		}
	}

	// send welcome message
	ircs.SendMessage(data.Nick, fmt.Sprintf("🎉 Your voice request in %s has been approved. Welcome! We'd love it if you'd take a moment to say hello and introduce yourself.", style.Bold(data.Channel)))

	logger.Infof(nil, "dashboard: approved voice request for %s in %s", data.Nick, data.Channel)
	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", nil)
}

func handleDashboardDenyVoiceRequest(ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	logger := log.Logger()

	if data.Nick == "" {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "nick is required", nil)
	}

	fs := firestore.Get()
	ch, err := fs.Channel(data.Channel)
	if err != nil || ch == nil {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "channel not found", nil)
	}

	found := false
	for _, vr := range ch.VoiceRequests {
		if vr.Nick == data.Nick {
			found = true
			break
		}
	}
	if !found {
		return models.NewDashboardResponseTask(data.RequestID, data.Action, false, "voice request not found", nil)
	}

	repository.RemoveChannelVoiceRequest(nil, ch, data.Nick, "")
	if err = repository.UpdateChannelVoiceRequests(nil, ch); err != nil {
		logger.Errorf(nil, "dashboard: error updating voice requests: %s", err)
	}

	logger.Infof(nil, "dashboard: denied voice request for %s in %s", data.Nick, data.Channel)
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

func handleDashboardListCommands(ctx context.Context, cfg *config.Config, ircs irc.IRC, data models.DashboardRequestTaskData) *models.Task {
	reg := commands.LoadCommandRegistry(ctx, cfg, ircs)
	return models.NewDashboardResponseTask(data.RequestID, data.Action, true, "", reg.CommandInfoList())
}
