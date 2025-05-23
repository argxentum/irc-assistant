package models

import "time"

func NewReconnectTask(dueAt time.Time) *Task {
	return newTask(TaskTypeReconnect, dueAt, map[string]any{})
}
