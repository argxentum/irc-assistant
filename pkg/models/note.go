package models

import (
	"assistant/pkg/api/text"
	"github.com/sqids/sqids-go"
	"time"
)

type Note struct {
	ID       string    `firestore:"id"`
	Content  string    `firestore:"content,omitempty"`
	Source   string    `firestore:"source,omitempty"`
	Keywords []string  `firestore:"keywords,omitempty"`
	NotedAt  time.Time `firestore:"noted_at"`
}

func NewNote(content, source string) *Note {
	s, _ := sqids.New()
	id, _ := s.Encode([]uint64{uint64(time.Now().Unix())})

	return &Note{
		ID:       id,
		Content:  content,
		Source:   source,
		NotedAt:  time.Now(),
		Keywords: text.ParseKeywords(content),
	}
}
