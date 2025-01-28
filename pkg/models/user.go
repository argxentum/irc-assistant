package models

import (
	"time"
)

const MaximumRecentUserMessages = 25

type User struct {
	Nick           string          `firestore:"nick"`
	Karma          int             `firestore:"karma"`
	RecentMessages []RecentMessage `firestore:"recent_messages"`
	CreatedAt      time.Time       `firestore:"created_at"`
	UpdatedAt      time.Time       `firestore:"updated_at"`
}

type RecentMessage struct {
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
