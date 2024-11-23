package models

import "time"

const ChannelInactivityTaskID = "channel-inactivity"

type PersistentTaskData struct {
	Channel string `firestore:"channel,omitempty" json:"channel,omitempty"`
}

// NewPersistentTask creates a persistent task. A persistent task is a task that is created once and then updated as needed. Creating a persistent task does not require the creation of a corresponding scheduled task.
func NewPersistentTask(id, channel, taskType string, dueAt time.Time) *Task {
	return &Task{
		ID:    id,
		Type:  taskType,
		DueAt: dueAt,
		Data: PersistentTaskData{
			Channel: channel,
		},
	}
}
