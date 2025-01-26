package firestore

import (
	"assistant/pkg/models"
	"fmt"
)

func (fs *Firestore) User(channel, nick string) (*models.User, error) {
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
