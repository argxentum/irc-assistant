package models

import "time"

type NotifyVoiceRequestsTaskData struct {
	Channel string `firestore:"channel" json:"channel"`
}

func NewNotifyVoiceRequestsTask(dueAt time.Time, channel string) *Task {
	return newTask(TaskTypeNotifyVoiceRequests, dueAt, NotifyVoiceRequestsTaskData{
		Channel: channel,
	})
}
