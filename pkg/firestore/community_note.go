package firestore

import (
	"assistant/pkg/models"
	"fmt"

	"cloud.google.com/go/firestore"
)

func (fs *Firestore) CommunityNote(channel, id string) (*models.CommunityNote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathNotes, id)
	return get[models.CommunityNote](fs.ctx, fs.client, path)
}

func (fs *Firestore) CommunityNotes(channel string) ([]*models.CommunityNote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathNotes)

	criteria := QueryCriteria{
		Path: path,
		OrderBy: []OrderBy{
			{
				Field:     "noted_at",
				Direction: firestore.Desc,
			},
		},
	}

	return query[models.CommunityNote](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) CommunityNoteForSource(channel, source string) (*models.CommunityNote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathNotes)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.AndFilter{
			Filters: []firestore.EntityFilter{
				firestore.PropertyFilter{
					Path:     "sources",
					Operator: ArrayContains,
					Value:    source,
				},
			},
		},
		OrderBy: []OrderBy{
			{
				Field:     "noted_at",
				Direction: firestore.Desc,
			},
		},
	}

	notes, err := query[models.CommunityNote](fs.ctx, fs.client, criteria)
	if err != nil {
		return nil, err
	}

	if len(notes) == 0 {
		return nil, nil
	}

	return notes[0], nil
}

func (fs *Firestore) CreateCommunityNote(channel string, note *models.CommunityNote) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathNotes, note.ID)
	return create(fs.ctx, fs.client, path, note)
}

func (fs *Firestore) SetCommunityNote(channel string, note *models.CommunityNote) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathNotes, note.ID)
	return set(fs.ctx, fs.client, path, note)
}

func (fs *Firestore) DeleteCommunityNote(channel, id string) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathNotes, id)
	return remove(fs.ctx, fs.client, path)
}
