package firestore

import (
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
	"fmt"
)

func (fs *Firestore) KarmaHistory(channel, nick string) ([]*models.KarmaHistory, error) {
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

	return query[models.KarmaHistory](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) SaveKarmaHistory(channel, nick string, kh *models.KarmaHistory) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathUsers, nick, pathKarmaHistory, kh.ID)
	return set(fs.ctx, fs.client, path, kh)
}
