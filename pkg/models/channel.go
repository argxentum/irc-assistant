package models

import (
	"time"
)

type Channel struct {
	Name               string    `firestore:"name" json:"name"`
	InactivityDuration string    `firestore:"inactivity_duration" json:"inactivity_duration"`
	CreatedAt          time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt          time.Time `firestore:"updated_at" json:"updated_at"`
}

func NewChannel(name string, inactivityDuration string) *Channel {
	return &Channel{
		Name:               name,
		InactivityDuration: inactivityDuration,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}
