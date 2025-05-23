package repository

import (
	"assistant/pkg/firestore"
	"assistant/pkg/models"
)

func GetOrCreateShortcut(url string) (*models.Shortcut, error) {
	fs := firestore.Get()

	shortcut := models.NewShortcut(url)

	existing, err := fs.Shortcut(shortcut.ID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// shortcut already exists
		return existing, nil
	}

	return shortcut, fs.CreateShortcut(shortcut)
}

func RemoveShortcut(id string) error {
	fs := firestore.Get()

	shortcut, err := fs.Shortcut(id)
	if err != nil {
		return err
	}

	if shortcut == nil {
		return nil
	}

	return fs.RemoveShortcut(shortcut.ID)
}
