package models

import "time"

type MuteRemovalTaskData struct {
	Nick      string `firestore:"nick" json:"nick"`
	Host      string `firestore:"host" json:"host"`
	Channel   string `firestore:"channel" json:"channel"`
	AutoVoice bool   `firestore:"auto_voice" json:"auto_voice"`
}

func NewMuteRemovalTask(dueAt time.Time, channel, nick, host string, autoVoice bool) *Task {
	return newTask(TaskTypeMuteRemoval, dueAt, MuteRemovalTaskData{
		Nick:      nick,
		Host:      host,
		Channel:   channel,
		AutoVoice: autoVoice,
	})
}
