package firestore

import (
	"assistant/pkg/models"
	"fmt"
)

func (fs *Firestore) Assistant() (*models.Assistant, error) {
	path := fmt.Sprintf("%s/%s", pathAssistants, fs.cfg.IRC.Nick)
	return get[models.Assistant](fs.ctx, fs.client, path)
}

func (fs *Firestore) CreateAssistant() error {
	path := fmt.Sprintf("%s/%s/", pathAssistants, fs.cfg.IRC.Nick)
	return create(fs.ctx, fs.client, path, models.NewAssistant(fs.cfg.IRC.Nick))
}
