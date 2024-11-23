package firestore

import (
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

func (fs *Firestore) BannedWords(channel string) ([]*models.BannedWord, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathBannedWords)
	return list[models.BannedWord](fs.ctx, fs.client, path)
}

func (fs *Firestore) AddBannedWord(channel, word string) error {
	id := fmt.Sprintf("%s-%s", models.PrefixBannedWord, uuid.NewString())
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathBannedWords, id)
	return create(fs.ctx, fs.client, path, &models.BannedWord{ID: id, Word: strings.ToLower(word)})
}

func (fs *Firestore) RemoveBannedWord(channel, word string) error {
	logger := log.Logger()
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathBannedWords)

	criteria := QueryCriteria{
		Path:   path,
		Filter: createPropertyFilter("word", Equal, strings.ToLower(word)),
	}

	bannedWords, err := query[models.BannedWord](fs.ctx, fs.client, criteria)
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
	return remove(fs.ctx, fs.client, path)
}

func (fs *Firestore) IsBannedWord(channel, word string) (bool, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathBannedWords)

	criteria := QueryCriteria{
		Path:   path,
		Filter: createPropertyFilter("word", Equal, strings.ToLower(word)),
	}

	ok, err := exists[models.BannedWord](fs.ctx, fs.client, criteria)
	return ok, err
}
