package models

import "time"

type MuteRemovalTaskData struct {
	Nick    string `firestore:"nick" json:"nick"`
	Channel string `firestore:"channel" json:"channel"`
}

func NewMuteRemovalTask(dueAt time.Time, nick, channel string) *Task {
	return newTask(TaskTypeMuteRemoval, dueAt, MuteRemovalTaskData{
		Nick:    nick,
		Channel: channel,
	})
}
