package models

import "time"

type Assistant struct {
	Name      string         `firestore:"name"`
	Cache     AssistantCache `firestore:"cache"`
	CreatedAt time.Time      `firestore:"created_at"`
	UpdatedAt time.Time      `firestore:"updated_at"`
}

type AssistantCache struct {
	BiasResults map[string]BiasResult `firestore:"bias_results"`
}

type BiasResult struct {
	Title       string    `json:"title"`
	Rating      string    `json:"rating"`
	Factual     string    `json:"factual"`
	Credibility string    `json:"credibility"`
	DetailURL   string    `json:"detail_url"`
	CachedAt    time.Time `firestore:"cached_at"`
}

func NewAssistant(name string) *Assistant {
	return &Assistant{
		Name:      name,
		Cache:     AssistantCache{BiasResults: make(map[string]BiasResult)},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
