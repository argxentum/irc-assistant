package firestore

import (
	"assistant/pkg/models"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
)

const pathStats = "stats"

func (fs *Firestore) AddChannelStats(channel string, stats *models.ChannelStats) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathStats)
	docID := fmt.Sprintf("%d", stats.Timestamp.Unix())
	docPath := fmt.Sprintf("%s/%s", path, docID)
	return set(fs.ctx, fs.client, docPath, stats)
}

func (fs *Firestore) GetChannelStats(channel string, since time.Time) ([]*models.ChannelStats, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathStats)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.PropertyFilter{
			Path:     "timestamp",
			Operator: GreaterThanOrEqual,
			Value:    since,
		},
		OrderBy: []OrderBy{
			{Field: "timestamp", Direction: firestore.Asc},
		},
	}

	return query[models.ChannelStats](fs.ctx, fs.client, criteria)
}
