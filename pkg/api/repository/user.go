package repository

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/firestore"
	"assistant/pkg/models"
	"fmt"
	"time"
)

const (
	OpAdd       = "+"
	OpIncrement = "++"
	OpSubtract  = "-"
	OpDecrement = "--"
)

func GetUser(e *irc.Event, channel, nick string, createIfNotExists bool) (*models.User, error) {
	fs := firestore.Get()
	u, err := fs.User(channel, nick)
	if err != nil {
		return nil, err
	}

	if u == nil && createIfNotExists {
		u = models.NewUser(nick)
		err = fs.CreateUser(channel, u)
		if err != nil {
			return nil, err
		}
	}

	return u, nil
}

func UpdateUserLastMessage(e *irc.Event, u *models.User) error {
	channel := e.ReplyTarget()
	u, err := GetUser(e, channel, u.Nick, false)
	if err != nil {
		return err
	}

	u.LastMessage = models.UserLastMessage{
		Message: e.Message(),
		At:      time.Now(),
	}

	fs := firestore.Get()
	return fs.UpdateUser(channel, u, map[string]interface{}{"last_message": u.LastMessage, "updated_at": time.Now()})
}

func IncrementUserKarma(e *irc.Event, u *models.User) error {
	u.Karma++
	fs := firestore.Get()
	return fs.UpdateUser(e.ReplyTarget(), u, map[string]interface{}{"karma": u.Karma, "updated_at": time.Now()})
}

func DecrementUserKarma(e *irc.Event, u *models.User) error {
	u.Karma--
	fs := firestore.Get()
	return fs.UpdateUser(e.ReplyTarget(), u, map[string]interface{}{"karma": u.Karma, "updated_at": time.Now()})
}

func AddUserKarmaHistory(e *irc.Event, channel, from, to, operation, reason string) (int, error) {
	u, err := GetUser(nil, channel, to, true)
	if err != nil {
		return 0, err
	}

	op := ""

	if operation == OpIncrement {
		op = OpAdd
		if err = IncrementUserKarma(e, u); err != nil {
			return 0, err
		}
	} else if operation == OpDecrement {
		op = OpSubtract
		if err = DecrementUserKarma(e, u); err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("invalid operation, %s", operation)
	}

	kh := models.NewKarmaHistory(from, op, 1, reason)
	return u.Karma, firestore.Get().SaveKarmaHistory(channel, to, kh)
}
