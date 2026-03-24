package firestore

import (
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
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

func (fs *Firestore) UpdateLLMResponse(id, content string) error {
	path := fmt.Sprintf("%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathLLMResponses, id)
	return update(fs.ctx, fs.client, path, map[string]any{
		"content":  content,
		"complete": true,
	})
}

func (fs *Firestore) LLMResponsesBySession(sessionID string) ([]*models.LLMResponse, error) {
	path := fmt.Sprintf("%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathLLMResponses)
	return query[models.LLMResponse](fs.ctx, fs.client, QueryCriteria{
		Path:   path,
		Filter: createPropertyFilter("session_id", Equal, sessionID),
		OrderBy: []OrderBy{
			{Field: "created_at", Direction: firestore.Asc},
		},
	})
}
