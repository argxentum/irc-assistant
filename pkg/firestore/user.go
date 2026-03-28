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

func (fs *Firestore) GetUsersByUserID(channel, userID string) ([]*models.User, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.PropertyFilter{
			Path:     "user_id",
			Operator: Equal,
			Value:    userID,
		},
	}

	return query[models.User](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) GetUsersByMask(channel, nick, userID, host string) ([]*models.User, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers)

	var filters []firestore.EntityFilter
	if nick != "" && nick != "*" {
		filters = append(filters, firestore.PropertyFilter{Path: "nick", Operator: Equal, Value: nick})
	}
	if userID != "" && userID != "*" {
		filters = append(filters, firestore.PropertyFilter{Path: "user_id", Operator: Equal, Value: userID})
	}
	if host != "" && host != "*" {
		filters = append(filters, firestore.PropertyFilter{Path: "host", Operator: Equal, Value: host})
	}

	if len(filters) == 0 {
		return nil, nil
	}

	criteria := QueryCriteria{Path: path}
	if len(filters) == 1 {
		criteria.Filter = filters[0]
	} else {
		criteria.Filter = firestore.AndFilter{Filters: filters}
	}

	return query[models.User](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) GetAllUsers(channel string) ([]*models.User, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers)
	return list[models.User](fs.ctx, fs.client, path)
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
