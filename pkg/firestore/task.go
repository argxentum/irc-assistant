package firestore

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/cloudtasks"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/firestore"
)

func (fs *Firestore) Task(path string) (*models.Task, error) {
	return get[models.Task](fs.ctx, fs.client, path)
}

func (fs *Firestore) SetTask(task *models.Task) error {
	return set(fs.ctx, fs.client, fs.TaskPath(task), task)
}

func (fs *Firestore) TaskPath(task *models.Task) string {
	switch task.Type {
	case models.TaskTypeReconnect:
		return fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathTasks, task.ID)
	case models.TaskTypeReminder:
		data := task.Data.(models.ReminderTaskData)
		return fmt.Sprintf("%s/%s", fs.tasksPath(data.User, data.Destination, task.Type), task.ID)
	case models.TaskTypeBanRemoval:
		data := task.Data.(models.BanRemovalTaskData)
		return fmt.Sprintf("%s/%s", fs.tasksPath("", data.Channel, task.Type), task.ID)
	case models.TaskTypeMuteRemoval:
		data := task.Data.(models.MuteRemovalTaskData)
		return fmt.Sprintf("%s/%s", fs.tasksPath("", data.Channel, task.Type), task.ID)
	case models.TaskTypeNotifyVoiceRequests:
		data := task.Data.(models.NotifyVoiceRequestsTaskData)
		return fmt.Sprintf("%s/%s", fs.tasksPath("", data.Channel, task.Type), task.ID)
	case models.TaskTypePersistentChannel, models.TaskTypePersistentChannelStats:
		data := task.Data.(models.PersistentTaskData)
		return fs.PersistentChannelTaskPath(data.Channel, task.ID)
	case models.TaskTypeDisinformationMutePenaltyRemoval:
		data := task.Data.(models.DisinformationMutePenaltyRemovalTaskData)
		return fmt.Sprintf("%s/%s", fs.tasksPath("", data.Channel, task.Type), task.ID)
	case models.TaskTypeDisinformationBanPenaltyRemoval:
		data := task.Data.(models.DisinformationBanPenaltyRemovalTaskData)
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
	case models.TaskTypeBanRemoval, models.TaskTypeMuteRemoval, models.TaskTypeNotifyVoiceRequests, models.TaskTypeDisinformationMutePenaltyRemoval, models.TaskTypeDisinformationBanPenaltyRemoval:
		return fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, destination, pathTasks)
	default:
		log.Logger().Errorf(nil, "can't create path for unknown task type: %s", taskType)
		return "unknown"
	}
}

func (fs *Firestore) AddTask(task *models.Task) error {
	logger := log.Logger()

	path := fs.TaskPath(task)
	logger.Debugf(nil, "creating task %s: %s", task.Type, path)

	if err := create(fs.ctx, fs.client, path, task); err != nil {
		logger.Warningf(nil, "error creating task, %s", err)
		return err
	}

	cloudTaskName, err := cloudtasks.Get().CreateTask(task)
	if err != nil {
		logger.Errorf(nil, "error creating cloud task %s: %s", task.ID, err)
		return err
	}

	task.CloudTaskName = cloudTaskName
	if err := update(fs.ctx, fs.client, path, map[string]any{"cloud_task_name": cloudTaskName}); err != nil {
		logger.Warningf(nil, "error storing cloud task name for %s: %s", task.ID, err)
	}

	return nil
}

func (fs *Firestore) CompleteTask(task *models.Task) error {
	logger := log.Logger()

	path := fs.TaskPath(task)
	logger.Debugf(nil, "completing task %s: %s", task.ID, path)

	if task.Status == models.TaskStatusCancelled && len(task.CloudTaskName) > 0 {
		if err := cloudtasks.Get().DeleteTask(task.CloudTaskName); err != nil {
			logger.Warningf(nil, "error deleting cloud task %s: %s", task.CloudTaskName, err)
		}
	}

	return update(fs.ctx, fs.client, path, map[string]interface{}{"status": task.Status, "runs": task.Runs})
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
		case models.TaskTypeMuteRemoval:
			var payload models.MuteRemovalTaskData
			err = json.Unmarshal(d, &payload)
			if err != nil {
				return nil, err
			}
			task.Data = payload
		case models.TaskTypeNotifyVoiceRequests:
			var payload models.NotifyVoiceRequestsTaskData
			err = json.Unmarshal(d, &payload)
			if err != nil {
				return nil, err
			}
			task.Data = payload
		}
	}

	return tasks, nil
}
