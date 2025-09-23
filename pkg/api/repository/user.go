package repository

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
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

func CreateUserFromNickChange(e *irc.Event, oldMask, newMask *irc.Mask) error {
	fs := firestore.Get()
	logger := log.Logger()

	logger.Debugf(e, "attempting to create user: %s", newMask.String())
	channels, err := GetAllChannels(e)
	if err != nil {
		return fmt.Errorf("error getting channels: %w", err)
	}

	for _, channel := range channels {
		logger.Debugf(e, "checking %s for %s", channel.Name, oldMask.String())

		oldUser, err := fs.GetUser(channel.Name, oldMask)
		if err != nil {
			return fmt.Errorf("error getting user: %w", err)
		}

		if oldUser != nil {
			logger.Debugf(e, "old user found %s, creating %s", oldMask.String(), newMask.String())
			newUser := models.NewUser(newMask)
			newUser.IsAutoVoiced = oldUser.IsAutoVoiced
			err = fs.CreateUser(channel.Name, newUser)
			if err != nil {
				return fmt.Errorf("error creating user: %w", err)
			}
			logger.Debugf(e, "created user: %s", newUser.Nick)
		}
	}

	return nil
}

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

func GetMostRecentUserKarmaHistoryFromSender(e *irc.Event, channel, recipient, sender string) (*models.KarmaHistory, error) {
	kh, err := firestore.Get().KarmaHistory(channel, recipient)
	if err != nil {
		return nil, err
	}

	if kh == nil {
		return nil, nil
	}

	for _, h := range kh {
		if h.From == sender {
			return h, nil
		}
	}

	return nil, nil
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

func GetPersonalNote(e *irc.Event, nick, id string) (*models.PersonalNote, error) {
	return firestore.Get().PersonalNote(nick, id)
}

func GetPersonalNotes(e *irc.Event, nick string) ([]*models.PersonalNote, error) {
	return firestore.Get().PersonalNotes(nick)
}

type personalNoteSearchResult struct {
	score int
	note  *models.PersonalNote
}

func GetPersonalNotesMatchingKeywords(e *irc.Event, nick string, keywords []string) ([]*models.PersonalNote, error) {
	matching, err := firestore.Get().PersonalNotesMatchingKeywords(nick, keywords)
	if err != nil {
		return nil, err
	}

	topMatches := make([]*models.PersonalNote, 0)

	sr := make([]personalNoteSearchResult, 0)
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
			sr = append(sr, personalNoteSearchResult{score, n})
		}
	}

	if len(topMatches) > 0 {
		return topMatches, nil
	}

	sort.Slice(sr, func(i, j int) bool {
		return sr[i].score > sr[j].score
	})

	notes := make([]*models.PersonalNote, 0)
	for _, r := range sr {
		notes = append(notes, r.note)
	}

	return notes, nil
}

func GetPersonalNotesMatchingSource(e *irc.Event, nick, source string) ([]*models.PersonalNote, error) {
	return firestore.Get().PersonalNotesMatchingSource(nick, source)
}

func AddPersonalNote(e *irc.Event, nick string, note *models.PersonalNote) error {
	return firestore.Get().CreatePersonalNote(nick, note)
}

func DeletePersonalNote(e *irc.Event, nick, id string) error {
	return firestore.Get().DeletePersonalNote(nick, id)
}
