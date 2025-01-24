package models

import (
	"assistant/pkg/api/irc"
	"slices"
	"strings"
	"time"
)

type Channel struct {
	Name                      string                     `firestore:"name" json:"name"`
	AutoVoiced                []string                   `firestore:"auto_voiced" json:"auto_voiced"`
	VoiceRequests             []VoiceRequest             `firestore:"voice_requests" json:"voice_requests"`
	IntroMessages             []string                   `firestore:"intro_messages" json:"intro_messages"`
	VoiceRequestNotifications []VoiceRequestNotification `firestore:"voice_request_notifications" json:"voice_request_notifications"`
	InactivityDuration        string                     `firestore:"inactivity_duration" json:"inactivity_duration"`
	Summarization             ChannelSummarization       `firestore:"summarization" json:"summarization"`
	CreatedAt                 time.Time                  `firestore:"created_at" json:"created_at"`
	UpdatedAt                 time.Time                  `firestore:"updated_at" json:"updated_at"`
}

type ChannelSummarization struct {
	DisinformationWarnings []string `firestore:"disinformation_warnings" json:"disinformation_warnings"`
}

type VoiceRequest struct {
	Nick        string    `firestore:"nick" json:"nick"`
	Username    string    `firestore:"username" json:"username"`
	Host        string    `firestore:"host" json:"host"`
	RequestedAt time.Time `firestore:"requested_at" json:"requested_at"`
}

type VoiceRequestNotification struct {
	User     string `firestore:"user" json:"user"`
	Interval string `firestore:"interval" json:"interval"`
}

func (vr VoiceRequest) Mask() string {
	mask := &irc.Mask{
		Nick:   vr.Nick,
		UserID: vr.Username,
		Host:   vr.Host,
	}

	return mask.String()
}

func (cs ChannelSummarization) IsPossibleDisinformation(url string) bool {
	return slices.ContainsFunc(cs.DisinformationWarnings, func(warning string) bool {
		return strings.HasPrefix(strings.ToLower(url), strings.ToLower(warning))
	})
}

func NewChannel(name string, inactivityDuration string) *Channel {
	return &Channel{
		Name:               name,
		AutoVoiced:         []string{},
		InactivityDuration: inactivityDuration,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}
