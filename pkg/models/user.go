package models

import (
	"assistant/pkg/api/irc"
	"time"
)

const MaximumRecentUserMessages = 25

type User struct {
	Nick           string          `firestore:"nick"`
	UserID         string          `firestore:"user_id"`
	Host           string          `firestore:"host"`
	Karma          int             `firestore:"karma"`
	Penalty        int             `firestore:"penalty"`
	Location       string          `firestore:"location"`
	IsAutoVoiced   bool            `firestore:"is_auto_voiced"`
	RecentMessages []RecentMessage `firestore:"recent_messages"`
	CreatedAt      time.Time       `firestore:"created_at"`
	UpdatedAt      time.Time       `firestore:"updated_at"`
}

type RecentMessage struct {
	Message string    `firestore:"message"`
	At      time.Time `firestore:"at"`
}

func NewUser(mask *irc.Mask) *User {
	return &User{
		Nick:      mask.Nick,
		UserID:    mask.UserID,
		Host:      mask.Host,
		Karma:     0,
		Penalty:   0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func NewUserWithNick(nick string) *User {
	return &User{
		Nick:      nick,
		Karma:     0,
		Penalty:   0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
