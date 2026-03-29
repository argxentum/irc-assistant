package models

import "time"

const ChannelStatsTaskID = "channel-stats"
const ChannelStatsInterval = 15 * time.Minute

type ChannelStats struct {
	TotalUsers   int       `firestore:"total_users" json:"total_users"`
	VoicedUsers  int       `firestore:"voiced_users" json:"voiced_users"`
	MessageCount int       `firestore:"message_count" json:"message_count"`
	Timestamp    time.Time `firestore:"timestamp" json:"timestamp"`
}
