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

func (fs *Firestore) DueTasks() ([]*models.Task, error) {
	logger := log.Logger()

	activeTasksPath := fmt.Sprintf("%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathTasks)
	criteria := QueryCriteria{
		Path: activeTasksPath,
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

	activeTasks, err := query[models.PendingTask](fs.ctx, fs.client, criteria)
	if err != nil {
		return nil, err
	}

	if len(activeTasks) == 0 {
		return nil, nil
	}

	logger.Debugf(nil, "tasks due: %d", len(activeTasks))

	tasks := make([]*models.Task, 0)

	for _, activeTask := range activeTasks {
		logger.Debugf(nil, "task due: %s", activeTask.Path)

		task, err := get[models.Task](fs.ctx, fs.client, activeTask.Path)
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
	pendingTask := models.NewPendingTask(task.ID, path, task.DueAt)
	pendingTaskPath := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathTasks, pendingTask.ID)
	if err := create[models.PendingTask](fs.ctx, fs.client, pendingTaskPath, pendingTask); err != nil {
		logger.Warningf(nil, "error creating task, %s", err)
		return err
	}

	return create(fs.ctx, fs.client, path, task)
}

func (fs *Firestore) RemoveTask(id, status string) error {
	logger := log.Logger()

	pendingTaskPath := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathTasks, id)
	pendingTask, err := get[models.PendingTask](fs.ctx, fs.client, pendingTaskPath)
	if err != nil {
		logger.Warningf(nil, "error getting active task, %s", err)
		return err
	}

	if err = remove(fs.ctx, fs.client, pendingTaskPath); err != nil {
		logger.Warningf(nil, "error removing active task, %s", err)
		return err
	}

	return update(fs.ctx, fs.client, pendingTask.Path, map[string]interface{}{"status": status})
}

func (fs *Firestore) GetPendingStatusTasks(user, destination, taskType string) ([]*models.Task, error) {
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
