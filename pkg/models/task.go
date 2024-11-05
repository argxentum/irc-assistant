package models

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
)

const taskIDPrefix = "task"

const (
	TaskTypeReminder   = "reminder"
	TaskTypeBanRemoval = "ban_removal"
)

const (
	TaskStatusPending   = "pending"
	TaskStatusComplete  = "complete"
	TaskStatusCancelled = "cancelled"
)

type PendingTask struct {
	ID    string    `firestore:"id" json:"id"`
	DueAt time.Time `firestore:"due_at" json:"due_at"`
	Path  string    `firestore:"path" json:"path"`
}

type Task struct {
	ID        string    `firestore:"id" json:"id"`
	Type      string    `firestore:"type" json:"type"`
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
	DueAt     time.Time `firestore:"due_at" json:"due_at"`
	Status    string    `firestore:"status" json:"status"`
	Data      any       `firestore:"data" json:"data"`
}

type ReminderTaskData struct {
	User        string `firestore:"user" json:"user"`
	Destination string `firestore:"destination" json:"destination"`
	Content     string `firestore:"content" json:"content"`
}

type BanRemovalTaskData struct {
	Mask    string `firestore:"mask" json:"mask"`
	Channel string `firestore:"channel" json:"channel"`
}

func NewPendingTask(id, path string, dueAt time.Time) *PendingTask {
	return &PendingTask{
		ID:    id,
		DueAt: dueAt,
		Path:  path,
	}
}

func NewReminderTask(dueAt time.Time, user, destination, content string) *Task {
	return newPendingTask(TaskTypeReminder, dueAt, ReminderTaskData{
		User:        user,
		Destination: destination,
		Content:     content,
	})
}

func NewBanRemovalTask(dueAt time.Time, mask, channel string) *Task {
	return newPendingTask(TaskTypeBanRemoval, dueAt, BanRemovalTaskData{
		Mask:    mask,
		Channel: channel,
	})
}

func newPendingTask(taskType string, due time.Time, payload any) *Task {
	return &Task{
		ID:        fmt.Sprintf("%s-%s", taskIDPrefix, uuid.NewString()),
		Type:      taskType,
		CreatedAt: time.Now(),
		DueAt:     due,
		Status:    TaskStatusPending,
		Data:      payload,
	}
}

func DeserializeTask(data []byte) (*Task, error) {
	var task Task
	err := json.Unmarshal(data, &task)
	if err != nil {
		return nil, err
	}

	d, err := json.Marshal(task.Data.(map[string]any))
	if err != nil {
		return nil, err
	}

	switch task.Type {
	case TaskTypeReminder:
		var payload ReminderTaskData
		err = json.Unmarshal(d, &payload)
		if err != nil {
			return nil, err
		}
		task.Data = payload
	case TaskTypeBanRemoval:
		var payload BanRemovalTaskData
		err = json.Unmarshal(d, &payload)
		if err != nil {
			return nil, err
		}
		task.Data = payload
	}

	return &task, nil
}

func (t *Task) Serialize() ([]byte, error) {
	return json.Marshal(t)
}
