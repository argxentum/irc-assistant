package repository

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"regexp"
	"strings"
	"time"
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

func UpdateAssistantCache(e *irc.Event, cache models.AssistantCache) error {
	log.Logger().Debugf(e, "updating assistant cache")
	fs := firestore.Get()
	return fs.UpdateAssistant(map[string]interface{}{"cache": cache})
}

func AddBiasResultToAssistantCache(e *irc.Event, input string, result models.BiasResult) error {
	log.Logger().Debugf(e, "adding bias result to assistant cache")
	assistant, err := GetAssistant(e, false)
	if err != nil {
		return err
	}

	if assistant.Cache.BiasResults == nil {
		assistant.Cache.BiasResults = make(map[string]models.BiasResult)
	}

	assistant.Cache.BiasResults[sanitizeInput(input)] = result

	return UpdateAssistantCache(e, assistant.Cache)
}

func GetBiasResultFromAssistantCache(e *irc.Event, input string) (models.BiasResult, bool) {
	assistant, err := GetAssistant(e, false)
	if err != nil {
		return models.BiasResult{}, false
	}

	result, ok := assistant.Cache.BiasResults[sanitizeInput(input)]

	if result.CachedAt.Before(time.Now().AddDate(-1, 0, 0)) {
		log.Logger().Debugf(e, "bias result for %s is stale, ignoring", input)
		return models.BiasResult{}, false
	}

	return result, ok
}

var rootDomainRegex = regexp.MustCompile(`(?:\.[a-z]+)+$`)

func sanitizeInput(input string) string {
	input = strings.ToLower(input)
	if rootDomainRegex.MatchString(input) {
		input = rootDomainRegex.ReplaceAllString(input, "")
	}
	return input
}
