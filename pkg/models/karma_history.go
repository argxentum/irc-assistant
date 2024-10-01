package models

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

type KarmaHistory struct {
	ID        string    `firestore:"id"`
	CreatedAt time.Time `firestore:"created_at"`
	To        string    `firestore:"to"`
	Op        string    `firestore:"op"`
	From      string    `firestore:"from"`
	Reason    string    `firestore:"reason"`
}

func NewKarmaHistory(to, from, op, reason string) *KarmaHistory {
	return &KarmaHistory{
		ID:        fmt.Sprintf("%s-%s", PrefixKarma, uuid.NewString()),
		CreatedAt: time.Now(),
		To:        to,
		Op:        op,
		From:      from,
		Reason:    reason,
	}
}
