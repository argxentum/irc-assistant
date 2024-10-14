package firestore

import (
	"assistant/pkg/api/context"
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
	"fmt"
)

const (
	OpIncrement = "++"
	OpDecrement = "--"
)

func (fs *Firestore) KarmaHistory(ctx context.Context, channel, nick string) ([]*models.KarmaHistory, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, nick, pathKarmaHistory)
	criteria := QueryCriteria{
		Path: path,
		OrderBy: []OrderBy{
			{
				Field:     "created_at",
				Direction: firestore.Desc,
			},
		},
	}

	return query[models.KarmaHistory](ctx, fs.client, criteria)
}

func (fs *Firestore) AddKarmaHistory(ctx context.Context, channel, from, to, op, reason string) (int, error) {
	u, err := fs.User(ctx, channel, to)
	if err != nil {
		return 0, err
	}
	if u == nil {
		u = models.NewUser(to, channel)
		err = fs.CreateUser(ctx, u)
		if err != nil {
			return 0, err
		}
	}

	if op == OpIncrement {
		u.Karma++
	} else if op == OpDecrement {
		u.Karma--
	} else {
		return 0, fmt.Errorf("invalid operation, %s", op)
	}

	if err = fs.UpdateUser(ctx, u); err != nil {
		return 0, err
	}

	karmaHistory := models.NewKarmaHistory(to, from, op, reason)
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, u.Nick, pathKarmaHistory, karmaHistory.ID)
	return u.Karma, create(ctx, fs.client, path, karmaHistory)
}
