package models

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

const disinformationIDPrefix = "disinfo"

type Disinformation struct {
	ID        string    `firestore:"id"`
	Source    string    `firestore:"source"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}

func NewDisinformation(source string) *Disinformation {
	return &Disinformation{
		ID:        fmt.Sprintf("%s-%s", disinformationIDPrefix, uuid.NewString()),
		Source:    source,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
