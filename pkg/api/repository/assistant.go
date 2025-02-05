package repository

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
)

func GetAssistant(e *irc.Event, createIfNotExists bool) (*models.Assistant, error) {
	logger := log.Logger()
	fs := firestore.Get()

	assistant, err := fs.Assistant()
	if err != nil {
		logger.Errorf(e, "error retrieving assistant, %s", err)
		return nil, err
	}

	if assistant == nil && createIfNotExists {
		logger.Debugf(e, "assistant not found, creating")
		assistant, err = fs.CreateAssistant()
		if err != nil {
			logger.Errorf(e, "error creating assistant, %s", err)
			return nil, err
		}
	}

	return assistant, nil
}
