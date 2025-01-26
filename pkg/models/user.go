package models

import (
	"time"
)

type User struct {
	Nick        string          `firestore:"nick"`
	Karma       int             `firestore:"karma"`
	LastMessage UserLastMessage `firestore:"last_message"`
	CreatedAt   time.Time       `firestore:"created_at"`
	UpdatedAt   time.Time       `firestore:"updated_at"`
}

type UserLastMessage struct {
	Message string    `firestore:"message"`
	At      time.Time `firestore:"at"`
}

func NewUser(nick string) *User {
	return &User{
		Nick:      nick,
		Karma:     0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
