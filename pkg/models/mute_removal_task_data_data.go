package models

import "time"

type MuteRemovalTaskData struct {
	Nick      string `firestore:"nick" json:"nick"`
	Channel   string `firestore:"channel" json:"channel"`
	AutoVoice bool   `firestore:"auto_voice" json:"auto_voice"`
}

func NewMuteRemovalTask(dueAt time.Time, nick, channel string, autoVoice bool) *Task {
	return newTask(TaskTypeMuteRemoval, dueAt, MuteRemovalTaskData{
		Nick:      nick,
		Channel:   channel,
		AutoVoice: autoVoice,
	})
}
