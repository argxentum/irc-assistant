package firestore

import (
	"assistant/pkg/models"
	"fmt"
)

func (fs *Firestore) Assistant() (*models.Assistant, error) {
	path := fmt.Sprintf("%s/%s", pathAssistants, fs.cfg.IRC.Nick)
	return get[models.Assistant](fs.ctx, fs.client, path)
}

func (fs *Firestore) CreateAssistant() (*models.Assistant, error) {
	path := fmt.Sprintf("%s/%s/", pathAssistants, fs.cfg.IRC.Nick)
	assistant := models.NewAssistant(fs.cfg.IRC.Nick)
	return assistant, create(fs.ctx, fs.client, path, assistant)
}

func (fs *Firestore) SetAssistant(assistant *models.Assistant) error {
	path := fmt.Sprintf("%s/%s", pathAssistants, fs.cfg.IRC.Nick)
	return set(fs.ctx, fs.client, path, assistant)
}

func (fs *Firestore) UpdateAssistant(fields map[string]any) error {
	path := fmt.Sprintf("%s/%s", pathAssistants, fs.cfg.IRC.Nick)
	return update(fs.ctx, fs.client, path, fields)
}
