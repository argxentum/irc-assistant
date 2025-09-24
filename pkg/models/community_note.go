package models

import (
	"time"

	"github.com/sqids/sqids-go"
)

type CommunityNote struct {
	ID             string    `firestore:"id"`
	Content        string    `firestore:"content,omitempty"`
	Author         string    `firestore:"author,omitempty"`
	Sources        []string  `firestore:"sources,omitempty"`
	CounterSources []string  `firestore:"counter_sources,omitempty"`
	NotedAt        time.Time `firestore:"noted_at"`
}

func NewCommunityNote(content, source, author string, counterSource ...string) *CommunityNote {
	s, _ := sqids.New()
	id, _ := s.Encode([]uint64{uint64(time.Now().Unix())})

	return &CommunityNote{
		ID:             id,
		Content:        content,
		Sources:        []string{source},
		CounterSources: counterSource,
		NotedAt:        time.Now(),
		Author:         author,
	}
}
