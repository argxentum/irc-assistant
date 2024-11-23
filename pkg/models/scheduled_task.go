package models

import (
	"time"
)

type ScheduledTask struct {
	ID    string    `firestore:"id" json:"id"`
	DueAt time.Time `firestore:"due_at" json:"due_at"`
	Path  string    `firestore:"path,omitempty" json:"path,omitempty"`
}

func NewScheduledTask(id, path string, dueAt time.Time) *ScheduledTask {
	return &ScheduledTask{
		ID:    id,
		DueAt: dueAt,
		Path:  path,
	}
}
