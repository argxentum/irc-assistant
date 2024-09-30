package firestore

import (
	"assistant/pkg/api/context"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

const pathAssistants = "assistants"
const pathChannels = "channels"
const pathBannedWords = "banned-words"
const prefixBannedWord = "banned-word"

func (fs *Firestore) AllBannedWords(ctx context.Context) ([]*models.BannedWord, error) {
	path := fmt.Sprintf("%s/%s/%s", pathAssistants, fs.cfg.Connection.Nick, pathBannedWords)
	return list[models.BannedWord](ctx, fs.client, path)
}

func (fs *Firestore) BannedWords(ctx context.Context, channel string) ([]*models.BannedWord, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.Connection.Nick, pathChannels, channel, pathBannedWords)
	return list[models.BannedWord](ctx, fs.client, path)
}

func (fs *Firestore) AddBannedWord(ctx context.Context, channel, word string) error {
	id := fmt.Sprintf("%s-%s", prefixBannedWord, uuid.NewString())
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.Connection.Nick, pathChannels, channel, pathBannedWords, id)
	return create(ctx, fs.client, path, &models.BannedWord{ID: id, Channel: strings.ToLower(channel), Word: strings.ToLower(word)})
}

func (fs *Firestore) RemoveBannedWord(ctx context.Context, channel, word string) error {
	logger := log.Logger()
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.Connection.Nick, pathChannels, channel, pathBannedWords)

	criteria := QueryCriteria{
		Path:   path,
		Filter: createPropertyFilter("word", Equal, strings.ToLower(word)),
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

	path = fmt.Sprintf("%s/%s", path, bannedWords[0].ID)
	return remove(ctx, fs.client, path)
}

func (fs *Firestore) IsBannedWord(ctx context.Context, channel, word string) (bool, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.Connection.Nick, pathChannels, channel, pathBannedWords)

	criteria := QueryCriteria{
		Path:   path,
		Filter: createPropertyFilter("word", Equal, strings.ToLower(word)),
	}

	ok, err := exists[models.BannedWord](ctx, fs.client, criteria)
	return ok, err
}
