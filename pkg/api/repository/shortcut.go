package repository

import (
	"assistant/pkg/firestore"
	"assistant/pkg/models"
	"fmt"
	"net/url"
)

const submissionURL = "https://archive.today/submit/?url=%s"

func GetShortcut(sourceURL, redirectURL string) (*models.Shortcut, error) {
	fs := firestore.Get()

	shortcut := models.NewShortcut(sourceURL, redirectURL)

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

func GetArchiveShortcutID(sourceURL string) (string, error) {
	redirectURL := fmt.Sprintf(submissionURL, url.QueryEscape(sourceURL))
	shortcut, err := GetShortcut(sourceURL, redirectURL)
	if err != nil {
		return "", err
	}

	if shortcut == nil {
		return "", nil
	}

	return shortcut.ID, nil
}

func GetShortcutSource(id string) (string, error) {
	fs := firestore.Get()

	shortcut, err := fs.Shortcut(id)
	if err != nil {
		return "", err
	}

	if shortcut == nil {
		return "", nil
	}

	return shortcut.SourceURL, nil
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
