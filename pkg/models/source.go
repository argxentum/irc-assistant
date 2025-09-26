package models

import (
	"time"

	"github.com/sqids/sqids-go"
)

type Source struct {
	ID          string    `firestore:"id"`
	Title       string    `firestore:"title"`
	Bias        string    `firestore:"bias"`
	Factuality  string    `firestore:"factuality"`
	Credibility string    `firestore:"credibility"`
	Reviews     []string  `firestore:"reviews"`
	URLs        []string  `firestore:"urls"`
	Paywall     bool      `firestore:"paywall"`
	Keywords    []string  `firestore:"keywords"`
	CreatedAt   time.Time `firestore:"created_at"`
	UpdatedAt   time.Time `firestore:"updated_at"`
}

func NewEmptySource() *Source {
	s, _ := sqids.New()
	id, _ := s.Encode([]uint64{uint64(time.Now().UnixNano())})

	return &Source{
		ID:        id,
		Reviews:   make([]string, 0),
		URLs:      make([]string, 0),
		Keywords:  make([]string, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func NewSource(title, bias, factuality, credibility, reviewURL string, urls, keywords []string) *Source {
	s, _ := sqids.New()
	id, _ := s.Encode([]uint64{uint64(time.Now().UnixNano())})

	references := make([]string, 0)
	if len(reviewURL) > 0 {
		references = append(references, reviewURL)
	}

	return &Source{
		ID:          id,
		Title:       title,
		URLs:        urls,
		Bias:        bias,
		Factuality:  factuality,
		Credibility: credibility,
		Reviews:     references,
		Keywords:    keywords,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
