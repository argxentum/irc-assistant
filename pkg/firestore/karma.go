package firestore

import (
	"assistant/pkg/api/context"
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	OpAdd       = "+"
	OpIncrement = "++"
	OpSubtract  = "-"
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

func (fs *Firestore) AddKarmaHistory(ctx context.Context, channel, from, to, operation, reason string) (int, error) {
	u, err := fs.User(ctx, channel, to)
	if status.Code(err) == codes.NotFound || u == nil {
		u = models.NewUser(to)
		err = fs.CreateUser(ctx, channel, u)
		if err != nil {
			return 0, err
		}
	}

	op := ""

	if operation == OpIncrement {
		op = OpAdd
		u.Karma++
	} else if operation == OpDecrement {
		op = OpSubtract
		u.Karma--
	} else {
		return 0, fmt.Errorf("invalid operation, %s", operation)
	}

	if err = fs.UpdateUser(ctx, channel, u); err != nil {
		return 0, err
	}

	karmaHistory := models.NewKarmaHistory(from, op, 1, reason)
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, u.Nick, pathKarmaHistory, karmaHistory.ID)
	return u.Karma, create(ctx, fs.client, path, karmaHistory)
}
