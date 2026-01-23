package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const taskIDPrefix = "task"

const (
	TaskTypeReminder                         = "reminder"
	TaskTypeBanRemoval                       = "ban_removal"
	TaskTypeMuteRemoval                      = "mute_removal"
	TaskTypeNotifyVoiceRequests              = "notify_voice_requests"
	TaskTypePersistentChannel                = "persistent_channel"
	TaskTypeReconnect                        = "reconnect"
	TaskTypeDisinformationMutePenaltyRemoval = "disinformation_penalty_removal"
	TaskTypeDisinformationBanPenaltyRemoval  = "disinformation_ban_penalty_removal"
)

const (
	TaskStatusPending   = "pending"
	TaskStatusComplete  = "complete"
	TaskStatusCancelled = "cancelled"
)

const ScheduledTaskMaxRuns = 3

type Task struct {
	ID        string    `firestore:"id,omitempty" json:"id,omitempty"`
	Type      string    `firestore:"type" json:"type"`
	Runs      int       `firestore:"runs,omitempty" json:"runs,omitempty"`
	CreatedAt time.Time `firestore:"created_at,omitempty" json:"created_at,omitempty"`
	DueAt     time.Time `firestore:"due_at" json:"due_at"`
	Status    string    `firestore:"status,omitempty" json:"status,omitempty"`
	Data      any       `firestore:"data,omitempty" json:"data,omitempty"`
}

func newTask(taskType string, due time.Time, payload any) *Task {
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
		if task.Data, err = deserializeTaskData[ReminderTaskData](d); err != nil {
			return nil, err
		}
	case TaskTypeBanRemoval:
		if task.Data, err = deserializeTaskData[BanRemovalTaskData](d); err != nil {
			return nil, err
		}
	case TaskTypeMuteRemoval:
		if task.Data, err = deserializeTaskData[MuteRemovalTaskData](d); err != nil {
			return nil, err
		}
	case TaskTypeNotifyVoiceRequests:
		if task.Data, err = deserializeTaskData[NotifyVoiceRequestsTaskData](d); err != nil {
			return nil, err
		}
	case TaskTypePersistentChannel:
		if task.Data, err = deserializeTaskData[PersistentTaskData](d); err != nil {
			return nil, err
		}
	case TaskTypeDisinformationMutePenaltyRemoval:
		if task.Data, err = deserializeTaskData[DisinformationMutePenaltyRemovalTaskData](d); err != nil {
			return nil, err
		}
	case TaskTypeDisinformationBanPenaltyRemoval:
		if task.Data, err = deserializeTaskData[DisinformationBanPenaltyRemovalTaskData](d); err != nil {
			return nil, err
		}
	}

	return &task, nil
}

func deserializeTaskData[T any](data []byte) (T, error) {
	var payload T
	err := json.Unmarshal(data, &payload)
	return payload, err
}

func (t *Task) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Task) IsDue() bool {
	return time.Now().After(t.DueAt)
}
