package models

import (
	"time"
)

type ScheduledTask struct {
	ID    string    `firestore:"id" json:"id"`
	Type  string    `firestore:"type,omitempty" json:"type,omitempty"`
	DueAt time.Time `firestore:"due_at" json:"due_at"`
	Path  string    `firestore:"path,omitempty" json:"path,omitempty"`
}

func NewScheduledTask(id, taskType, path string, dueAt time.Time) *ScheduledTask {
	return &ScheduledTask{
		ID:    id,
		Type:  taskType,
		DueAt: dueAt,
		Path:  path,
	}
}
