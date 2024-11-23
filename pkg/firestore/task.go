package firestore

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
	"encoding/json"
	"fmt"
	"time"
)

func (fs *Firestore) Task(path string) (*models.Task, error) {
	return get[models.Task](fs.ctx, fs.client, path)
}

func (fs *Firestore) SetTask(task *models.Task) error {
	return set(fs.ctx, fs.client, fs.taskPath(task), task)
}

func (fs *Firestore) DueTasks() ([]*models.Task, error) {
	logger := log.Logger()

	scheduledTasksPath := fmt.Sprintf("%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathTasks)
	criteria := QueryCriteria{
		Path: scheduledTasksPath,
		Filter: firestore.PropertyFilter{
			Path:     "due_at",
			Operator: LessThanOrEqual,
			Value:    time.Now(),
		},
		OrderBy: []OrderBy{
			{
				Field:     "due_at",
				Direction: firestore.Asc,
			},
		},
	}

	scheduledTasks, err := query[models.ScheduledTask](fs.ctx, fs.client, criteria)
	if err != nil {
		return nil, err
	}

	if len(scheduledTasks) == 0 {
		return nil, nil
	}

	logger.Debugf(nil, "tasks due: %d", len(scheduledTasks))

	tasks := make([]*models.Task, 0)

	for _, scheduledTask := range scheduledTasks {
		logger.Debugf(nil, "task due: %s", scheduledTask.Path)

		task, err := get[models.Task](fs.ctx, fs.client, scheduledTask.Path)
		if err != nil {
			logger.Warningf(nil, "error getting task, %s", err)
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (fs *Firestore) taskPath(task *models.Task) string {
	switch task.Type {
	case models.TaskTypeReminder:
		data := task.Data.(models.ReminderTaskData)
		return fmt.Sprintf("%s/%s", fs.tasksPath(data.User, data.Destination, task.Type), task.ID)
	case models.TaskTypeBanRemoval:
		data := task.Data.(models.BanRemovalTaskData)
		return fmt.Sprintf("%s/%s", fs.tasksPath("", data.Channel, task.Type), task.ID)
	case models.TaskTypePersistentChannel:
		data := task.Data.(models.PersistentTaskData)
		return fs.PersistentChannelTaskPath(data.Channel, task.ID)
	}
	return "unknown"
}

func (fs *Firestore) tasksPath(user, destination, taskType string) string {
	switch taskType {
	case models.TaskTypeReminder:
		if !irc.IsChannel(destination) {
			return fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, user, pathTasks)
		} else {
			return fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, destination, pathUsers, user, pathTasks)
		}
	case models.TaskTypeBanRemoval:
		return fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, destination, pathTasks)
	default:
		return "unknown"
	}
}

func (fs *Firestore) AddTask(task *models.Task) error {
	logger := log.Logger()

	path := fs.taskPath(task)
	scheduled := models.NewScheduledTask(task.ID, path, task.DueAt)
	scheduledPath := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathTasks, scheduled.ID)
	if err := create[models.ScheduledTask](fs.ctx, fs.client, scheduledPath, scheduled); err != nil {
		logger.Warningf(nil, "error creating task, %s", err)
		return err
	}

	return create(fs.ctx, fs.client, path, task)
}

func (fs *Firestore) RemoveScheduledTaskAndUpdateTask(task *models.Task) error {
	logger := log.Logger()

	scheduledPath := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathTasks, task.ID)
	scheduled, err := get[models.ScheduledTask](fs.ctx, fs.client, scheduledPath)
	if err != nil {
		logger.Warningf(nil, "error getting scheduled task, %s", err)
		return err
	}

	if scheduled == nil {
		return nil
	}

	if err = remove(fs.ctx, fs.client, scheduledPath); err != nil {
		logger.Warningf(nil, "error removing scheduled task, %s", err)
		return err
	}

	return update(fs.ctx, fs.client, scheduled.Path, map[string]interface{}{"status": task.Status, "runs": task.Runs})
}

func (fs *Firestore) GetPendingTasks(user, destination, taskType string) ([]*models.Task, error) {
	path := fs.tasksPath(user, destination, taskType)

	criteria := QueryCriteria{
		Path: path,
		OrderBy: []OrderBy{
			{
				Field:     "due_at",
				Direction: firestore.Asc,
			},
		},
		Filter: firestore.AndFilter{
			Filters: []firestore.EntityFilter{
				createPropertyFilter("type", Equal, taskType),
				createPropertyFilter("status", Equal, models.TaskStatusPending),
			},
		},
	}

	tasks, err := query[models.Task](fs.ctx, fs.client, criteria)
	if err != nil {
		return nil, err
	}

	return fs.populateTaskData(tasks)
}

func (fs *Firestore) populateTaskData(tasks []*models.Task) ([]*models.Task, error) {
	for _, task := range tasks {
		d, err := json.Marshal(task.Data.(map[string]any))
		if err != nil {
			return nil, err
		}

		switch task.Type {
		case models.TaskTypeReminder:
			var payload models.ReminderTaskData
			err = json.Unmarshal(d, &payload)
			if err != nil {
				return nil, err
			}
			task.Data = payload
		case models.TaskTypeBanRemoval:
			var payload models.BanRemovalTaskData
			err = json.Unmarshal(d, &payload)
			if err != nil {
				return nil, err
			}
			task.Data = payload
		}
	}

	return tasks, nil
}
