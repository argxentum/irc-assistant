package firestore

import (
	"assistant/pkg/api/context"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

const CollectionBannedWords = "banned-words"
const prefixBannedWord = "banned-word"

func (fs *Firestore) AllBannedWords(ctx context.Context) ([]*models.BannedWord, error) {
	return query[models.BannedWord](ctx, fs.client, QueryCriteria{Path: CollectionBannedWords})
}

func (fs *Firestore) BannedWords(ctx context.Context, channel string) ([]*models.BannedWord, error) {
	criteria := QueryCriteria{
		Path: CollectionBannedWords,
		Filter: firestore.AndFilter{
			Filters: []firestore.EntityFilter{
				createPropertyFilter("bot", Equal, fs.cfg.Connection.Nick),
				createPropertyFilter("server", Equal, fs.cfg.Connection.ServerName),
				createPropertyFilter("channel", Equal, channel),
			},
		},
	}

	return query[models.BannedWord](ctx, fs.client, criteria)
}

func (fs *Firestore) AddBannedWord(ctx context.Context, channel, word string) error {
	id := fmt.Sprintf("%s-%s", prefixBannedWord, uuid.NewString())
	path := fmt.Sprintf("%s/%s", CollectionBannedWords, id)
	return create(ctx, fs.client, path, &models.BannedWord{ID: id, Bot: fs.cfg.Connection.Nick, Server: fs.cfg.Connection.ServerName, Channel: channel, Word: strings.ToLower(word)})
}

func (fs *Firestore) RemoveBannedWord(ctx context.Context, channel, word string) error {
	logger := log.Logger()

	criteria := QueryCriteria{
		Path: CollectionBannedWords,
		Filter: firestore.AndFilter{
			Filters: []firestore.EntityFilter{
				createPropertyFilter("bot", Equal, fs.cfg.Connection.Nick),
				createPropertyFilter("server", Equal, fs.cfg.Connection.ServerName),
				createPropertyFilter("channel", Equal, channel),
				createPropertyFilter("word", Equal, strings.ToLower(word)),
			},
		},
	}

	bannedWords, err := query[models.BannedWord](ctx, fs.client, criteria)
	if err != nil {
		logger.Rawf(log.Warning, "error querying banned words, %s", err)
		return err
	}

	if len(bannedWords) == 0 {
		logger.Rawf(log.Debug, "no matching banned words found")
		return nil
	}

	logger.Rawf(log.Debug, "removing %s", bannedWords[0].ID)

	path := fmt.Sprintf("%s/%s", CollectionBannedWords, bannedWords[0].ID)
	return remove(ctx, fs.client, path)
}

func (fs *Firestore) IsBannedWord(ctx context.Context, channel, word string) (bool, error) {
	criteria := QueryCriteria{
		Path: CollectionBannedWords,
		Filter: firestore.AndFilter{
			Filters: []firestore.EntityFilter{
				createPropertyFilter("bot", Equal, fs.cfg.Connection.Nick),
				createPropertyFilter("server", Equal, fs.cfg.Connection.ServerName),
				createPropertyFilter("channel", Equal, channel),
				createPropertyFilter("word", Equal, strings.ToLower(word)),
			},
		},
	}

	ok, err := exists[models.BannedWord](ctx, fs.client, criteria)
	return ok, err
}
