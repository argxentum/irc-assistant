package models

import "time"

type BanRemovalTaskData struct {
	Mask    string `firestore:"mask" json:"mask"`
	Channel string `firestore:"channel" json:"channel"`
}

func NewBanRemovalTask(dueAt time.Time, mask, channel string) *Task {
	return newTask(TaskTypeBanRemoval, dueAt, BanRemovalTaskData{
		Mask:    mask,
		Channel: channel,
	})
}
