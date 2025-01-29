package models

import (
	"assistant/pkg/api/style"
	"fmt"
	"strings"
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

	ratingColor := style.ColorNone
	if strings.Contains(strings.ToLower(br.Rating), "least biased") {
		ratingColor = style.ColorGreen
	} else if strings.ToLower(br.Rating) == "left" || strings.ToLower(br.Rating) == "right" {
		ratingColor = style.ColorYellow
	} else if strings.Contains(strings.ToLower(br.Rating), "extreme") {
		ratingColor = style.ColorRed
	}

	if len(br.Rating) > 0 {
		desc += fmt.Sprintf("%s: %s", style.Underline("Bias"), style.ColorForeground(br.Rating, ratingColor))
	}

	factualColor := style.ColorNone
	if strings.Contains(strings.ToLower(br.Factual), "high") {
		factualColor = style.ColorGreen
	} else if strings.ToLower(br.Factual) == "mixed" {
		factualColor = style.ColorYellow
	} else if strings.Contains(strings.ToLower(br.Factual), "low") {
		factualColor = style.ColorRed
	}

	if len(br.Factual) > 0 {
		if len(desc) > 0 {
			desc += ", "
		}
		desc += fmt.Sprintf("%s: %s", style.Underline("Factual reporting"), style.ColorForeground(br.Factual, factualColor))
	}

	credibilityColor := style.ColorNone
	if strings.Contains(strings.ToLower(br.Credibility), "high") {
		credibilityColor = style.ColorGreen
	} else if strings.Contains(strings.ToLower(br.Credibility), "medium") {
		credibilityColor = style.ColorYellow
	} else if strings.Contains(strings.ToLower(br.Credibility), "low") {
		credibilityColor = style.ColorRed
	}

	if len(br.Credibility) > 0 {
		if len(desc) > 0 {
			desc += ", "
		}
		desc += fmt.Sprintf("%s: %s", style.Underline("Credibility"), style.ColorForeground(br.Credibility, credibilityColor))
	}

	if len(desc) > 0 {
		desc = fmt.Sprintf("📊 %s %s", style.Bold(br.Title), desc)
	}

	return desc
}
