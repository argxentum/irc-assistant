package firestore

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
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

	activeTasks, err := query[models.ActiveTask](fs.ctx, fs.client, criteria)
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
		if !irc.IsChannel(data.Destination) && data.Destination == data.User {
			return fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, data.User, pathTasks, task.ID)
		} else {
			return fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, data.Destination, pathUsers, data.User, pathTasks, task.ID)
		}
	case models.TaskTypeBanRemoval:
		data := task.Data.(models.BanRemovalTaskData)
		return fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, data.Channel, pathTasks, task.ID)
	default:
		return "unknown"
	}
}

func (fs *Firestore) AddTask(task *models.Task) error {
	logger := log.Logger()

	path := fs.taskPath(task)
	activeTask := models.NewActiveTask(task.ID, path, task.DueAt)
	activeTaskPath := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathTasks, activeTask.ID)
	if err := create[models.ActiveTask](fs.ctx, fs.client, activeTaskPath, activeTask); err != nil {
		logger.Warningf(nil, "error creating task, %s", err)
		return err
	}

	return create(fs.ctx, fs.client, path, task)
}

func (fs *Firestore) RemoveTask(id string) error {
	logger := log.Logger()

	activeTaskPath := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathTasks, id)
	activeTask, err := get[models.ActiveTask](fs.ctx, fs.client, activeTaskPath)
	if err != nil {
		logger.Warningf(nil, "error getting active task, %s", err)
		return err
	}

	if err = remove(fs.ctx, fs.client, activeTaskPath); err != nil {
		logger.Warningf(nil, "error removing active task, %s", err)
		return err
	}

	return update(fs.ctx, fs.client, activeTask.Path, map[string]interface{}{"status": models.TaskStatusComplete})
}
