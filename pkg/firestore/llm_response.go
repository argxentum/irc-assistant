package firestore

import (
	"assistant/pkg/models"
	"fmt"
)

func (fs *Firestore) CreateLLMResponse(r *models.LLMResponse) error {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathLLMResponses, r.ID)
	return create(fs.ctx, fs.client, path, r)
}

func (fs *Firestore) LLMResponse(id string) (*models.LLMResponse, error) {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathLLMResponses, id)
	return get[models.LLMResponse](fs.ctx, fs.client, path)
}
