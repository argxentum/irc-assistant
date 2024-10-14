package models

import (
	"time"
)

type User struct {
	Nick      string    `firestore:"nick"`
	Channel   string    `firestore:"channel"`
	Karma     int       `firestore:"karma"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}

func NewUser(nick, channel string) *User {
	return &User{
		Nick:      nick,
		Channel:   channel,
		Karma:     0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
