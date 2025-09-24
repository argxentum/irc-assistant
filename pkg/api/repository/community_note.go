package repository

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/firestore"
	"assistant/pkg/models"
)

func CommunityNote(e *irc.Event, channel string, id string) (*models.CommunityNote, error) {
	return firestore.Get().CommunityNote(channel, id)
}

func GetCommunityNoteForSource(e *irc.Event, channel, source string) (*models.CommunityNote, error) {
	return firestore.Get().CommunityNoteForSource(channel, source)
}

func CreateCommunityNote(e *irc.Event, channel string, note *models.CommunityNote) error {
	return firestore.Get().CreateCommunityNote(channel, note)
}

func UpdateCommunityNote(e *irc.Event, channel string, note *models.CommunityNote) error {
	return firestore.Get().SetCommunityNote(channel, note)
}

func DeleteCommunityNote(e *irc.Event, channel, id string) error {
	return firestore.Get().DeleteCommunityNote(channel, id)
}
