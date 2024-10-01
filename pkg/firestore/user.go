package firestore

import (
	"assistant/pkg/api/context"
	"assistant/pkg/models"
	"fmt"
)

func (fs *Firestore) User(ctx context.Context, channel, nick string) (*models.User, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.Connection.Nick, pathChannels, channel, pathUsers)
	criteria := createQueryCriteria(path, "nick", Equal, nick)
	users, err := query[models.User](ctx, fs.client, criteria)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}

func (fs *Firestore) CreateUser(ctx context.Context, user *models.User) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.Connection.Nick, pathChannels, user.Channel, pathUsers, user.ID)
	return create(ctx, fs.client, path, user)
}

func (fs *Firestore) UpdateUser(ctx context.Context, user *models.User) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.Connection.Nick, pathChannels, user.Channel, pathUsers, user.ID)
	return set(ctx, fs.client, path, user)
}
