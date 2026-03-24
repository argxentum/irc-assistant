package models

import (
	"time"

	"github.com/google/uuid"
)

type LLMResponse struct {
	ID        string    `firestore:"id" json:"id"`
	TaskID    string    `firestore:"task_id" json:"task_id"`
	SessionID string    `firestore:"session_id" json:"session_id"`
	Channel   string    `firestore:"channel" json:"channel"`
	Nick      string    `firestore:"nick" json:"nick"`
	Prompt    string    `firestore:"prompt" json:"prompt"`
	Content   string    `firestore:"content" json:"content"`
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
}

func NewLLMResponse(taskID, sessionID, channel, nick, prompt, content string) *LLMResponse {
	return &LLMResponse{
		ID:        uuid.NewString(),
		TaskID:    taskID,
		SessionID: sessionID,
		Channel:   channel,
		Nick:      nick,
		Prompt:    prompt,
		Content:   content,
		CreatedAt: time.Now(),
	}
}
