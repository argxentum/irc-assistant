package firestore

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
	"fmt"
)

func (fs *Firestore) GetUser(channel string, mask *irc.Mask) (*models.User, error) {
	users, err := fs.GetAllMatchingUsers(channel, mask)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}

	return users[0], nil
}

func (fs *Firestore) GetAllMatchingUsers(channel string, mask *irc.Mask) ([]*models.User, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.OrFilter{
			Filters: []firestore.EntityFilter{
				firestore.PropertyFilter{
					Path:     "nick",
					Operator: Equal,
					Value:    mask.Nick,
				},
				firestore.PropertyFilter{
					Path:     "host",
					Operator: Equal,
					Value:    mask.Host,
				},
			},
		},
	}

	return query[models.User](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) GetUsersByHost(channel, host string) ([]*models.User, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.PropertyFilter{
			Path:     "host",
			Operator: Equal,
			Value:    host,
		},
	}

	return query[models.User](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) GetUserByNick(channel, nick string) (*models.User, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, nick)
	return get[models.User](fs.ctx, fs.client, path)
}

func (fs *Firestore) CreateUser(channel string, user *models.User) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, user.Nick)
	return create(fs.ctx, fs.client, path, user)
}

func (fs *Firestore) SetUser(channel string, user *models.User) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, user.Nick)
	return set(fs.ctx, fs.client, path, user)
}

func (fs *Firestore) UpdateUser(channel string, user *models.User, fields map[string]any) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, user.Nick)
	return update(fs.ctx, fs.client, path, fields)
}
