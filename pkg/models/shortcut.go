package models

import (
	"github.com/sqids/sqids-go"
	"hash/fnv"
)

type Shortcut struct {
	ID  string `firestore:"id"`
	URL string `firestore:"url"`
}

func NewShortcut(url string) *Shortcut {
	s, _ := sqids.New()
	h := fnv.New64()
	_, _ = h.Write([]byte(url))
	id, _ := s.Encode([]uint64{h.Sum64()})

	return &Shortcut{
		ID:  id,
		URL: url,
	}
}
