package models

import "time"

type ReminderTaskData struct {
	User        string `firestore:"user" json:"user"`
	Destination string `firestore:"destination" json:"destination"`
	Content     string `firestore:"content" json:"content"`
}

func NewReminderTask(dueAt time.Time, user, destination, content string) *Task {
	return newTask(TaskTypeReminder, dueAt, ReminderTaskData{
		User:        user,
		Destination: destination,
		Content:     content,
	})
}
