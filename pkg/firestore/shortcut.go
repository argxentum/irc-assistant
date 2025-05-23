package firestore

import (
	"assistant/pkg/models"
	"fmt"
)

func (fs *Firestore) Shortcut(id string) (*models.Shortcut, error) {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathShortcuts, id)
	return get[models.Shortcut](fs.ctx, fs.client, path)
}

func (fs *Firestore) Shortcuts() ([]*models.Shortcut, error) {
	path := fmt.Sprintf("%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathShortcuts)
	return list[models.Shortcut](fs.ctx, fs.client, path)
}

func (fs *Firestore) CreateShortcut(shortcut *models.Shortcut) error {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathShortcuts, shortcut.ID)
	return create(fs.ctx, fs.client, path, shortcut)
}

func (fs *Firestore) RemoveShortcut(id string) error {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathShortcuts, id)
	return remove(fs.ctx, fs.client, path)
}
