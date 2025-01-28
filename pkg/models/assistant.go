package models

import (
	"assistant/pkg/api/style"
	"fmt"
	"time"
)

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

func (br BiasResult) Description() string {
	desc := ""

	if len(br.Rating) > 0 {
		desc += fmt.Sprintf("%s: %s", style.Underline("Bias"), br.Rating)
	}

	if len(br.Factual) > 0 {
		if len(desc) > 0 {
			desc += ", "
		}
		desc += fmt.Sprintf("%s: %s", style.Underline("Factual reporting"), br.Factual)
	}

	if len(br.Credibility) > 0 {
		if len(desc) > 0 {
			desc += ", "
		}
		desc += fmt.Sprintf("%s: %s", style.Underline("Credibility"), br.Credibility)
	}

	if len(desc) > 0 {
		desc = fmt.Sprintf("📊 %s %s", style.Bold(br.Title), desc)
	}

	return desc
}
