package models

import "time"

type Assistant struct {
	Name      string    `firestore:"name" json:"name"`
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at" json:"updated_at"`
}

func NewAssistant(name string) *Assistant {
	return &Assistant{
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
