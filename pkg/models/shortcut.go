package models

import (
	"github.com/sqids/sqids-go"
	"hash/fnv"
)

type Shortcut struct {
	ID          string `firestore:"id"`
	SourceURL   string `firestore:"source_url"`
	RedirectURL string `firestore:"redirect_url"`
}

func NewShortcut(sourceURL, redirectURL string) *Shortcut {
	s, _ := sqids.New()
	h := fnv.New64()
	_, _ = h.Write([]byte(redirectURL))
	id, _ := s.Encode([]uint64{h.Sum64()})

	return &Shortcut{
		ID:          id,
		RedirectURL: redirectURL,
		SourceURL:   sourceURL,
	}
}
