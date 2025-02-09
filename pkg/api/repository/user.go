package repository

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/firestore"
	"assistant/pkg/models"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	OpAdd       = "+"
	OpIncrement = "++"
	OpSubtract  = "-"
	OpDecrement = "--"
)

func GetUsersByHost(e *irc.Event, channel, host string) ([]*models.User, error) {
	fs := firestore.Get()
	return fs.GetUsersByHost(channel, host)
}

func GetUserByNick(e *irc.Event, channel, nick string, createIfNotExists bool) (*models.User, error) {
	fs := firestore.Get()
	u, err := fs.GetUserByNick(channel, nick)
	if err != nil {
		return nil, err
	}

	if u == nil && createIfNotExists {
		u = models.NewUserWithNick(nick)
		err = fs.CreateUser(channel, u)
		if err != nil {
			return nil, err
		}
	}

	return u, nil
}

func GetUserByMask(e *irc.Event, channel string, mask *irc.Mask, createIfNotExists bool) (*models.User, error) {
	u, err := firestore.Get().GetUser(channel, mask)
	if err != nil {
		return nil, err
	}

	if u == nil && createIfNotExists {
		u = models.NewUser(mask)
		err = firestore.Get().CreateUser(channel, u)
		if err != nil {
			return nil, err
		}
	}

	return u, nil
}

func AddRecentUserMessage(e *irc.Event, u *models.User) error {
	channel := e.ReplyTarget()

	u.RecentMessages = append(u.RecentMessages, models.RecentMessage{
		Message: e.Message(),
		At:      time.Now(),
	})

	if len(u.RecentMessages) > models.MaximumRecentUserMessages {
		u.RecentMessages = u.RecentMessages[1:]
	}

	fs := firestore.Get()
	return fs.UpdateUser(channel, u, map[string]interface{}{"recent_messages": u.RecentMessages, "updated_at": time.Now()})
}

func FindMostRecentUserMessage(e *irc.Event, u *models.User) (models.RecentMessage, bool) {
	if len(u.RecentMessages) == 0 {
		return models.RecentMessage{}, false
	}

	return u.RecentMessages[len(u.RecentMessages)-1], true
}

func FindRecentUserMessage(e *irc.Event, u *models.User, input string) (models.RecentMessage, bool) {
	input = strings.TrimSpace(input)

	if len(u.RecentMessages) == 0 {
		return models.RecentMessage{}, false
	}

	// iterate through recent messages backward, to match the most recent message
	for i := len(u.RecentMessages) - 1; i >= 0; i-- {
		m := u.RecentMessages[i]
		if strings.Contains(strings.TrimSpace(strings.ToLower(m.Message)), input) {
			return m, true
		}
	}

	return models.RecentMessage{}, false
}

func UpdateUserIsAutoVoiced(e *irc.Event, channel string, u *models.User) error {
	fs := firestore.Get()
	return fs.UpdateUser(channel, u, map[string]interface{}{"is_auto_voiced": u.IsAutoVoiced, "updated_at": time.Now()})
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
	u, err := GetUserByNick(nil, channel, to, true)
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

func GetUserNote(e *irc.Event, nick, id string) (*models.Note, error) {
	return firestore.Get().UserNote(nick, id)
}

func GetUserNotes(e *irc.Event, nick string) ([]*models.Note, error) {
	return firestore.Get().UserNotes(nick)
}

type noteSearchResult struct {
	score int
	note  *models.Note
}

func GetUserNotesMatchingKeywords(e *irc.Event, nick string, keywords []string) ([]*models.Note, error) {
	matching, err := firestore.Get().UserNotesMatchingKeywords(nick, keywords)
	if err != nil {
		return nil, err
	}

	topMatches := make([]*models.Note, 0)

	sr := make([]noteSearchResult, 0)
	for _, n := range matching {
		score := 0
		allMatch := true
		for _, k := range keywords {
			if strings.Contains(strings.ToLower(n.Content), k) {
				score++
			} else {
				allMatch = false
			}
		}

		if allMatch {
			topMatches = append(topMatches, n)
		}

		if score > 0 {
			sr = append(sr, noteSearchResult{score, n})
		}
	}

	if len(topMatches) > 0 {
		return topMatches, nil
	}

	sort.Slice(sr, func(i, j int) bool {
		return sr[i].score > sr[j].score
	})

	notes := make([]*models.Note, 0)
	for _, r := range sr {
		notes = append(notes, r.note)
	}

	return notes, nil
}

func GetUserNotesMatchingSource(e *irc.Event, nick, source string) ([]*models.Note, error) {
	return firestore.Get().UserNotesMatchingSource(nick, source)
}

func AddUserNote(e *irc.Event, nick string, note *models.Note) error {
	return firestore.Get().CreateUserNote(nick, note)
}

func DeleteUserNote(e *irc.Event, nick, id string) error {
	return firestore.Get().DeleteUserNote(nick, id)
}
