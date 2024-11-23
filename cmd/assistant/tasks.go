package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
	"strings"
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

const inactivityPosts = 1

func processPersistentChannel(ctx context.Context, cfg *config.Config, irc irc.IRC, task *models.Task) error {
	logger := log.Logger()
	fs := firestore.Get()
	logger.Debugf(nil, "processing persistent channel task for %s", task.ID)

	switch task.ID {
	case models.ChannelInactivityTaskID:
		posts, err := reddit.RisingSubredditPosts(ctx, cfg, "politics", inactivityPosts)
		if err != nil {
			logger.Errorf(nil, "error getting top subreddit posts, %s", err)
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

		message := fmt.Sprintf("%s of inactivity detected, sharing top rising post in r/politics...", elapse.ParseDurationIntoPlainEnglish(channel.InactivityDuration))
		if len(posts) > 1 {
			message = fmt.Sprintf("%s of inactivity detected, sharing top %d rising posts in r/politics...", elapse.ParseDurationIntoPlainEnglish(channel.InactivityDuration), inactivityPosts)
		}
		irc.SendMessage(channelName, message)

		time.Sleep(5 * time.Second)

		for _, post := range posts {
			messages := make([]string, 0)
			messages = append(messages, style.Bold(post.Post.Title))
			messages = append(messages, post.Post.URL)

			comment := sanitize(post.Comment.Body)

			if post.Comment != nil {
				messages = append(messages, fmt.Sprintf("Top comment (u/%s): %s", post.Comment.Author, style.Italics(comment)))
			}

			irc.SendMessages(channelName, messages)

			time.Sleep(3 * time.Second)
		}
	default:
		return fmt.Errorf("unknown persistent channel task, %s", task.ID)
	}

	return nil
}

const commentMaxLength = 256

func sanitize(s string) string {
	// replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")

	// collapse multiple spaces
	s = strings.Join(strings.Fields(s), " ")

	// trim leading and trailing spaces
	s = strings.TrimSpace(s)

	// truncate to max length
	if len(s) > commentMaxLength {
		s = s[:commentMaxLength] + "..."
	}
	return s
}
