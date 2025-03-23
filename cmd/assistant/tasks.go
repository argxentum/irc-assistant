package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
	"slices"
	"time"
)

func processTasks(ctx context.Context, cfg *config.Config, irc irc.IRC) {
	fs := firestore.Get()
	logger := log.Logger()

	go func() {
		err := queue.Get().Receive(func(task *models.Task) {
			logger.Debugf(nil, "received task %s: %s [%d runs]", task.ID, task.Type, task.Runs)

			var isScheduledTask bool
			var err error

			switch task.Type {
			case models.TaskTypeReminder:
				isScheduledTask = true
				err = processReminder(irc, task)
			case models.TaskTypeBanRemoval:
				isScheduledTask = true
				err = processBanRemoval(irc, task)
			case models.TaskTypeMuteRemoval:
				isScheduledTask = true
				err = processMuteRemoval(irc, task)
			case models.TaskTypeNotifyVoiceRequests:
				isScheduledTask = true
				err = processNotifyVoiceRequests(irc, task)
			case models.TaskTypePersistentChannel:
				isScheduledTask = false
				err = processPersistentChannel(ctx, cfg, irc, task)
			}

			task.Runs++

			if isScheduledTask {
				if err != nil {
					if task.Runs >= models.ScheduledTaskMaxRuns {
						task.Status = models.TaskStatusCancelled
					} else {
						task.Status = models.TaskStatusPending
					}
				} else {
					task.Status = models.TaskStatusComplete
				}

				err = fs.RemoveScheduledTaskAndUpdateTask(task)
				if err != nil {
					logger.Errorf(nil, "error completing %s, %s", task.ID, err)
				}
			} else {
				channel, err := fs.Channel(task.Data.(models.PersistentTaskData).Channel)
				if err != nil {
					logger.Errorf(nil, "error getting channel for %s, %s", task.ID, err)
					return
				}

				if channel == nil {
					logger.Errorf(nil, "channel %s not found for %s", task.Data.(models.PersistentTaskData).Channel, task.ID)
					return
				}

				duration, err := elapse.ParseDuration(channel.InactivityDuration)
				if err != nil {
					logger.Errorf(nil, "error parsing duration %s, %s", channel.InactivityDuration, err)
					return
				}

				task.DueAt = time.Now().Add(duration)
				err = fs.SetTask(task)
				if err != nil {
					logger.Errorf(nil, "error updating %s, %s", task.ID, err)
				}
			}
		})

		if err != nil {
			logger.Errorf(nil, "error processing due tasks, %s", err)
		}
	}()
}

func processReminder(ircs irc.IRC, task *models.Task) error {
	data := task.Data.(models.ReminderTaskData)

	logger := log.Logger()
	logger.Debugf(nil, "processing reminder for %s: %s", data.User, data.Content)

	message := ""
	if irc.IsChannel(data.Destination) {
		message = fmt.Sprintf("%s: here's the reminder you set %s: %s", data.User, elapse.TimeDescription(task.CreatedAt), style.Bold(data.Content))
	} else {
		message = fmt.Sprintf("Here's the reminder you set %s: %s", elapse.TimeDescription(task.CreatedAt), style.Bold(data.Content))
	}

	ircs.SendMessage(data.Destination, message)

	return nil
}

func processBanRemoval(irc irc.IRC, task *models.Task) error {
	data := task.Data.(models.BanRemovalTaskData)

	logger := log.Logger()
	logger.Debugf(nil, "processing ban removal for %s in %s", data.Mask, data.Channel)

	irc.Unban(data.Channel, data.Mask)

	return nil
}

func processMuteRemoval(irc irc.IRC, task *models.Task) error {
	data := task.Data.(models.MuteRemovalTaskData)

	logger := log.Logger()
	logger.Debugf(nil, "processing mute removal for %s in %s", data.Nick, data.Channel)

	users := make([]*models.User, 0)

	// find user by nick
	if len(data.Nick) > 0 {
		u, err := repository.GetUserByNick(nil, data.Channel, data.Nick, false)
		if err != nil {
			return fmt.Errorf("error getting user by nick: %v", err)
		}

		if u != nil {
			users = append(users, u)
		}
	}

	// find users with matching host
	if len(data.Host) > 0 {
		us, err := repository.GetUsersByHost(nil, data.Channel, data.Host)
		if err != nil {
			return fmt.Errorf("error getting users by host: %v", err)
		}

		for _, u := range us {
			if u.Nick != data.Nick {
				users = append(users, u)
			}
		}
	}

	for _, u := range users {
		irc.Voice(data.Channel, u.Nick)
		logger.Debugf(nil, "unmuted %s in %s", u.Nick, data.Channel)

		if data.AutoVoice {
			fs := firestore.Get()
			u.IsAutoVoiced = true
			if err := fs.UpdateUser(data.Channel, u, map[string]interface{}{"is_auto_voiced": u.IsAutoVoiced, "updated_at": time.Now()}); err != nil {
				return fmt.Errorf("error updating user isAutoVoiced, %s", err)
			}
			logger.Debugf(nil, "auto-voiced %s in %s", u.Nick, data.Channel)
		}
	}

	return nil
}

func processNotifyVoiceRequests(irc irc.IRC, task *models.Task) error {
	data := task.Data.(models.NotifyVoiceRequestsTaskData)

	logger := log.Logger()
	logger.Debugf(nil, "processing notify voice requests in %s", data.Channel)

	ch, err := repository.GetChannel(nil, data.Channel)
	if err != nil {
		return fmt.Errorf("error retrieving channel, %s", err)
	}

	if len(ch.VoiceRequestNotifications) == 0 {
		logger.Debugf(nil, "no voice request notifications configured for %s", data.Channel)
		return nil
	}

	if len(ch.VoiceRequests) == 0 {
		logger.Debugf(nil, "no voice requests in %s", data.Channel)
		return nil
	}

	name := "requests"
	if len(ch.VoiceRequests) == 1 {
		name = "request"
	}

	notice := fmt.Sprintf("Note: %s outstanding voice %s in %s. To review: %s", style.Bold(fmt.Sprintf("%d", len(ch.VoiceRequests))), name, style.Bold(data.Channel), style.Italics(fmt.Sprintf("!vr %s", data.Channel)))
	for _, n := range ch.VoiceRequestNotifications {
		irc.SendMessage(n.User, notice)
	}
	return nil
}

const inactivityPostsBuffer = 3

var previousInactivityPostURLs = make([]string, 0)

func processPersistentChannel(ctx context.Context, cfg *config.Config, irc irc.IRC, task *models.Task) error {
	logger := log.Logger()
	fs := firestore.Get()
	logger.Debugf(nil, "processing persistent channel task for %s", task.ID)

	switch task.ID {
	case models.ChannelInactivityTaskID:
		posts, err := reddit.SubredditCategoryPostsWithTopComment(ctx, cfg, cfg.IRC.Inactivity.Subreddit, cfg.IRC.Inactivity.Category, cfg.IRC.Inactivity.Posts+inactivityPostsBuffer)
		if err != nil {
			logger.Errorf(nil, "error getting subreddit category posts, %s", err)
			return err
		}

		channelName := task.Data.(models.PersistentTaskData).Channel
		if len(channelName) == 0 {
			return fmt.Errorf("channel name is empty")
		}

		channel, err := fs.Channel(channelName)
		if err != nil {
			logger.Errorf(nil, "error getting channel, %s", err)
		}

		if channel == nil {
			logger.Errorf(nil, "channel %s does not exist, exiting", channelName)
			return nil
		}

		filteredPosts := make([]reddit.PostWithTopComment, 0)
		for _, post := range posts {
			if slices.Contains(previousInactivityPostURLs, post.Post.URL) {
				logger.Debugf(nil, "skipping duplicate post %s", post.Post.URL)
				continue
			}

			filteredPosts = append(filteredPosts, post)
			if len(filteredPosts) == cfg.IRC.Inactivity.Posts {
				break
			}
		}

		if len(filteredPosts) == 0 {
			logger.Debugf(nil, "no inactivity posts found for channel %s matching filter requirements", channelName)
			return nil
		}

		message := fmt.Sprintf("🕑 %s of inactivity, sharing a %s post from r/%s:", elapse.ParseDurationDescription(channel.InactivityDuration), cfg.IRC.Inactivity.Category, cfg.IRC.Inactivity.Subreddit)
		if len(filteredPosts) > 1 {
			message = fmt.Sprintf("🕑 %s of inactivity, sharing %d %s posts from r/%s:", elapse.ParseDurationDescription(channel.InactivityDuration), len(filteredPosts), cfg.IRC.Inactivity.Category, cfg.IRC.Inactivity.Subreddit)
		}
		irc.SendMessage(channelName, message)
		time.Sleep(1 * time.Second)

		for i, post := range filteredPosts {
			messages := make([]string, 0)
			messages = append(messages, post.Post.FormattedTitle())
			messages = append(messages, post.Post.URL)
			previousInactivityPostURLs = append(previousInactivityPostURLs, post.Post.URL)

			if post.Comment != nil {
				messages = append(messages, post.Comment.FormattedBody())
			}

			source, err := repository.FindSource(post.Post.URL)
			if err != nil {
				logger.Errorf(nil, "error finding source, %s", err)
			}

			if source != nil {
				messages = append(messages, repository.ShortSourceSummary(source))
			}

			irc.SendMessages(channelName, messages)
			logger.Debugf(nil, "shared r/%s post \"%s\" in %s due to inactivity", cfg.IRC.Inactivity.Subreddit, post.Post.Title, channelName)

			if i < len(filteredPosts)-1 {
				time.Sleep(3 * time.Second)
			}
		}
	default:
		return fmt.Errorf("unknown persistent channel task, %s", task.ID)
	}

	return nil
}
