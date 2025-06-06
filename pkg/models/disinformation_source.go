package models

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

const PrefixDisinformationSource = "disinformation-source"

type DisinformationSource struct {
	ID        string    `firestore:"id"`
	Source    string    `firestore:"source"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}

func NewDisinformationSource(source string) *DisinformationSource {
	return &DisinformationSource{
		ID:        fmt.Sprintf("%s-%s", PrefixDisinformationSource, uuid.NewString()),
		Source:    source,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
