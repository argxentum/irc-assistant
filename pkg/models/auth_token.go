package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

const AuthTokenLength = 32 // 32 bytes = 64 hex chars
const AuthTokenExpiry = 5 * time.Minute

type AuthToken struct {
	Token     string    `firestore:"token"`
	Nick      string    `firestore:"nick"`
	Channel   string    `firestore:"channel"`
	Used      bool      `firestore:"used"`
	CreatedAt time.Time `firestore:"created_at"`
	ExpiresAt time.Time `firestore:"expires_at"`
}

func NewAuthToken(nick, channel string) (*AuthToken, error) {
	b := make([]byte, AuthTokenLength)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	now := time.Now()
	return &AuthToken{
		Token:     hex.EncodeToString(b),
		Nick:      nick,
		Channel:   channel,
		Used:      false,
		CreatedAt: now,
		ExpiresAt: now.Add(AuthTokenExpiry),
	}, nil
}

func (t *AuthToken) IsValid() bool {
	return !t.Used && time.Now().Before(t.ExpiresAt)
}
