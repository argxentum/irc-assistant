package firestore

import (
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"time"
)

func (fs *Firestore) PersistentTasks(taskIDs ...string) ([]*models.Task, error) {
	logger := log.Logger()
	tasks := make([]*models.Task, 0)

	channels, err := fs.Channels()
	if err != nil {
		logger.Debugf(nil, "error getting channels, %s", err)
	}

	for _, channel := range channels {
		for _, taskID := range taskIDs {
			path := fs.PersistentChannelTaskPath(channel.Name, taskID)
			task, err := fs.Task(path)
			if err != nil {
				logger.Debugf(nil, "error getting persistent task at %s, %s", path, err)
				continue
			}
			if task == nil || !task.IsDue() {
				continue
			}
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (fs *Firestore) PersistentChannelTaskPath(channel, id string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathTasks, id)
}

func (fs *Firestore) SetPersistentChannelTaskDue(channel, id string, duration time.Duration) error {
	path := fs.PersistentChannelTaskPath(channel, id)
	task, err := fs.Task(path)
	if err != nil {
		return err
	}

	if task == nil {
		task = models.NewPersistentTask(id, channel, models.TaskTypePersistentChannel, time.Now().Add(duration))
		return create[models.Task](fs.ctx, fs.client, path, task)
	}

	return update(fs.ctx, fs.client, path, map[string]any{"due_at": time.Now().Add(duration), "updated_at": time.Now()})
}
