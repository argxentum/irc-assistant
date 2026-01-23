package models

import "time"

type DisinformationMutePenaltyRemovalTaskData struct {
	Channel string `firestore:"channel" json:"channel"`
	Nick    string `firestore:"nick" json:"nick"`
	Penalty int    `firestore:"penalty" json:"penalty"`
}

func NewDisinformationMutePenaltyRemovalTask(dueAt time.Time, channel, nick string, penalty int) *Task {
	return newTask(TaskTypeDisinformationMutePenaltyRemoval, dueAt, DisinformationMutePenaltyRemovalTaskData{
		Channel: channel,
		Nick:    nick,
		Penalty: penalty,
	})
}
