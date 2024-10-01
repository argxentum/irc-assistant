package models

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID        string    `firestore:"id"`
	Nick      string    `firestore:"nick"`
	Channel   string    `firestore:"channel"`
	Karma     int       `firestore:"karma"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}

func NewUser(nick, channel string) *User {
	return &User{
		ID:        fmt.Sprintf("%s-%s", PrefixUser, uuid.NewString()),
		Nick:      nick,
		Channel:   channel,
		Karma:     0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
