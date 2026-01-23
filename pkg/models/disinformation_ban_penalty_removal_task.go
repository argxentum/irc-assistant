package models

import "time"

type DisinformationBanPenaltyRemovalTaskData struct {
	Channel string `firestore:"channel" json:"channel"`
	Nick    string `firestore:"nick" json:"nick"`
	Penalty int    `firestore:"penalty" json:"penalty"`
}

func NewDisinformationBanPenaltyRemovalTask(dueAt time.Time, channel, nick string, penalty int) *Task {
	return newTask(TaskTypeDisinformationBanPenaltyRemoval, dueAt, DisinformationBanPenaltyRemovalTaskData{
		Channel: channel,
		Nick:    nick,
		Penalty: penalty,
	})
}
