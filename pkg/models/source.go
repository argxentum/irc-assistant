package models

import "time"

type Source struct {
	ID          string    `firestore:"id"`
	Title       string    `firestore:"title"`
	Bias        string    `firestore:"bias"`
	Factuality  string    `firestore:"factuality"`
	Credibility string    `firestore:"credibility"`
	Reviews     []string  `firestore:"reviews"`
	URLs        []string  `firestore:"urls"`
	Keywords    []string  `firestore:"keywords"`
	CreatedAt   time.Time `firestore:"created_at"`
	UpdatedAt   time.Time `firestore:"updated_at"`
}

func NewSource(id, title, bias, factuality, credibility, reviewURL string, urls, keywords []string) *Source {
	return &Source{
		ID:          id,
		Title:       title,
		URLs:        urls,
		Bias:        bias,
		Factuality:  factuality,
		Credibility: credibility,
		Reviews:     []string{reviewURL},
		Keywords:    keywords,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
