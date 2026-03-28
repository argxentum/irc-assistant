package firestore

import (
	"assistant/pkg/models"
	"fmt"
)

func (fs *Firestore) CreateAuthToken(token *models.AuthToken) error {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathAuthTokens, token.Token)
	return create(fs.ctx, fs.client, path, token)
}

func (fs *Firestore) GetAuthToken(token string) (*models.AuthToken, error) {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathAuthTokens, token)
	return get[models.AuthToken](fs.ctx, fs.client, path)
}

func (fs *Firestore) MarkAuthTokenUsed(token string) error {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathAuthTokens, token)
	return update(fs.ctx, fs.client, path, map[string]any{"used": true})
}
