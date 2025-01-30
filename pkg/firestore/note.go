package firestore

import (
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
	"fmt"
)

func (fs *Firestore) UserNote(nick, id string) (*models.Note, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes, id)
	return get[models.Note](fs.ctx, fs.client, path)
}

func (fs *Firestore) UserNotes(nick string) ([]*models.Note, error) {
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

	return query[models.Note](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) UserNotesMatchingKeywords(nick string, keywords []string) ([]*models.Note, error) {
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

	return query[models.Note](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) UserNotesMatchingSource(nick, source string) ([]*models.Note, error) {
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

	return query[models.Note](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) CreateUserNote(nick string, note *models.Note) error {
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

func (fs *Firestore) SetUserNote(nick string, note *models.Note) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes, note.ID)
	return set(fs.ctx, fs.client, path, note)
}

func (fs *Firestore) DeleteUserNote(nick, id string) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUsers, nick, pathNotes, id)
	return remove(fs.ctx, fs.client, path)
}
