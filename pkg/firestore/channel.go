package firestore

import (
	"assistant/pkg/models"
	"fmt"
)

func (fs *Firestore) Channels() ([]*models.Channel, error) {
	path := fmt.Sprintf("%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels)
	return list[models.Channel](fs.ctx, fs.client, path)
}

func (fs *Firestore) Channel(channel string) (*models.Channel, error) {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel)
	return get[models.Channel](fs.ctx, fs.client, path)
}

func (fs *Firestore) UpdateChannel(channel string, fields map[string]any) error {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel)
	return update(fs.ctx, fs.client, path, fields)
}

func (fs *Firestore) CreateChannel(channel *models.Channel) error {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel.Name)
	return create(fs.ctx, fs.client, path, channel)
}
