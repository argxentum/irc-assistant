package models

import "time"

type DisinformationPenaltyRemovalTaskData struct {
	Channel string `firestore:"channel" json:"channel"`
	Nick    string `firestore:"nick" json:"nick"`
	Penalty int    `firestore:"penalty" json:"penalty"`
}

func NewDisinformationPenaltyRemovalTask(dueAt time.Time, channel, nick string, penalty int) *Task {
	return newTask(TaskTypeDisinformationPenaltyRemoval, dueAt, DisinformationPenaltyRemovalTaskData{
		Channel: channel,
		Nick:    nick,
		Penalty: penalty,
	})
}
