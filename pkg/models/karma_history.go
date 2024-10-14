package models

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

type KarmaHistory struct {
	ID        string    `firestore:"id"`
	CreatedAt time.Time `firestore:"created_at"`
	Op        string    `firestore:"op"`
	Quantity  int       `firestore:"quantity"`
	From      string    `firestore:"from"`
	Reason    string    `firestore:"reason,omitempty"`
}

func NewKarmaHistory(from, op string, quantity int, reason string) *KarmaHistory {
	return &KarmaHistory{
		ID:        fmt.Sprintf("%s-%s", PrefixKarma, uuid.NewString()),
		CreatedAt: time.Now(),
		From:      from,
		Op:        op,
		Quantity:  quantity,
		Reason:    reason,
	}
}
