package models

import (
	"slices"
	"strings"
	"time"
)

type Channel struct {
	Name               string               `firestore:"name" json:"name"`
	InactivityDuration string               `firestore:"inactivity_duration" json:"inactivity_duration"`
	Summarization      ChannelSummarization `firestore:"summarization" json:"summarization"`
	CreatedAt          time.Time            `firestore:"created_at" json:"created_at"`
	UpdatedAt          time.Time            `firestore:"updated_at" json:"updated_at"`
}

type ChannelSummarization struct {
	DisinformationWarnings []string `firestore:"disinformation_warnings" json:"disinformation_warnings"`
}

func (cs ChannelSummarization) IsPossibleDisinformation(url string) bool {
	return slices.ContainsFunc(cs.DisinformationWarnings, func(warning string) bool {
		return strings.HasPrefix(strings.ToLower(url), strings.ToLower(warning))
	})
}

func NewChannel(name string, inactivityDuration string) *Channel {
	return &Channel{
		Name:               name,
		InactivityDuration: inactivityDuration,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}
