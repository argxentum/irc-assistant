package firestore

import (
	"assistant/pkg/models"
	"fmt"

	"cloud.google.com/go/firestore"
)

func (fs *Firestore) IncrementCommandUsage(channel, commandName string) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathCommandUsage, commandName)
	doc := fs.client.Doc(path)
	_, err := doc.Set(fs.ctx, map[string]any{
		"name":  commandName,
		"count": firestore.Increment(1),
	}, firestore.MergeAll)
	return err
}

func (fs *Firestore) ListCommandUsage(channel string) ([]*models.CommandUsage, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathCommandUsage)
	return list[models.CommandUsage](fs.ctx, fs.client, path)
}
