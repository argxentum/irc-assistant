package firestore

import (
	"assistant/pkg/models"
	"fmt"

	"cloud.google.com/go/firestore"
)

func (fs *Firestore) PersonalNote(nick, id string) (*models.PersonalNote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes, id)
	return get[models.PersonalNote](fs.ctx, fs.client, path)
}

func (fs *Firestore) PersonalNotes(nick string) ([]*models.PersonalNote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes)

	criteria := QueryCriteria{
		Path: path,
		OrderBy: []OrderBy{
			{
				Field:     "noted_at",
				Direction: firestore.Desc,
			},
		},
	}

	return query[models.PersonalNote](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) PersonalNotesMatchingKeywords(nick string, keywords []string) ([]*models.PersonalNote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.PropertyFilter{
			Path:     "keywords",
			Operator: ArrayContainsAny,
			Value:    keywords,
		},
		OrderBy: []OrderBy{
			{
				Field:     "noted_at",
				Direction: firestore.Desc,
			},
		},
	}

	return query[models.PersonalNote](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) PersonalNotesMatchingSource(nick, source string) ([]*models.PersonalNote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.AndFilter{
			Filters: []firestore.EntityFilter{
				firestore.PropertyFilter{
					Path:     "source",
					Operator: GreaterThanOrEqual,
					Value:    source,
				},
				firestore.PropertyFilter{
					Path:     "source",
					Operator: LessThan,
					Value:    source + "\ufffd",
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

	return query[models.PersonalNote](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) CreatePersonalNote(nick string, note *models.PersonalNote) error {
	usersPath := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick)
	user, err := get[models.User](fs.ctx, fs.client, usersPath)
	if err != nil {
		return err
	}
	if user == nil {
		empty := make(map[string]any)
		err = create[map[string]any](fs.ctx, fs.client, usersPath, &empty)
		if err != nil {
			return err
		}
	}

	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes, note.ID)
	return create(fs.ctx, fs.client, path, note)
}

func (fs *Firestore) SetPersonalNote(nick string, note *models.PersonalNote) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes, note.ID)
	return set(fs.ctx, fs.client, path, note)
}

func (fs *Firestore) DeletePersonalNote(nick, id string) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes, id)
	return remove(fs.ctx, fs.client, path)
}
