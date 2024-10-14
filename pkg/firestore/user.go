package firestore

import (
	"assistant/pkg/api/context"
	"assistant/pkg/models"
	"fmt"
)

func (fs *Firestore) User(ctx context.Context, channel, nick string) (*models.User, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, nick)
	return get[models.User](ctx, fs.client, path)
}

func (fs *Firestore) CreateUser(ctx context.Context, channel string, user *models.User) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, user.Nick)
	return create(ctx, fs.client, path, user)
}

func (fs *Firestore) UpdateUser(ctx context.Context, channel string, user *models.User) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, user.Nick)
	return set(ctx, fs.client, path, user)
}
