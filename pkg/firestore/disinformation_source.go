package firestore

import (
	"assistant/pkg/log"
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
	"fmt"
	"slices"
	"strings"
)

var disinformationSources map[string][]string

func (fs *Firestore) pathToDisinformationSources(channel string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathDisinformationSources)
}

func (fs *Firestore) pathToDisinformationSource(channel string, source *models.DisinformationSource) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathDisinformationSources, source.ID)
}

func (fs *Firestore) DisinformationSources(channel string) ([]*models.DisinformationSource, error) {
	return list[models.DisinformationSource](fs.ctx, fs.client, fs.pathToDisinformationSources(channel))
}

func (fs *Firestore) DisinformationSource(channel, source string) (*models.DisinformationSource, error) {
	criteria := QueryCriteria{
		Path: fs.pathToDisinformationSources(channel),
		Filter: firestore.PropertyFilter{
			Path:     "source",
			Operator: Equal,
			Value:    strings.TrimSpace(strings.ToLower(source)),
		},
	}

	sources, err := query[models.DisinformationSource](fs.ctx, fs.client, criteria)
	if err != nil {
		return nil, err
	}

	if len(sources) == 0 {
		return nil, nil
	}

	return sources[0], nil
}

func (fs *Firestore) AddDisinformationSource(channel, source string) error {
	logger := log.Logger()
	source = strings.TrimSpace(strings.ToLower(source))

	if fs.IsDisinformationSource(channel, source) {
		logger.Debugf(nil, "disinformation source already exists: %s", source)
		return nil
	}

	if disinformationSources == nil {
		disinformationSources = make(map[string][]string)
	}
	if len(disinformationSources[channel]) == 0 {
		disinformationSources[channel] = make([]string, 0)
	}
	disinformationSources[channel] = append(disinformationSources[channel], source)

	ds := models.NewDisinformationSource(source)
	return create(fs.ctx, fs.client, fs.pathToDisinformationSource(channel, ds), ds)
}

func (fs *Firestore) DeleteDisinformationSource(channel, source string) error {
	logger := log.Logger()
	source = strings.TrimSpace(strings.ToLower(source))

	if !fs.IsDisinformationSource(channel, source) {
		logger.Debugf(nil, "disinformation source does not exist: %s", source)
		return nil
	}

	disinformationSources[channel] = slices.DeleteFunc(disinformationSources[channel], func(s string) bool {
		return s == source
	})

	ds, err := fs.DisinformationSource(channel, source)
	if err != nil {
		logger.Errorf(nil, "error checking disinformation source %s: %v", source, err)
	}

	return remove(fs.ctx, fs.client, fs.pathToDisinformationSource(channel, ds))
}

func (fs *Firestore) IsDisinformationSource(channel, source string) bool {
	if disinformationSources == nil {
		if err := fs.ReloadDisinformationSources(channel); err != nil {
			log.Logger().Errorf(nil, "failed to reload disinformation sources for channel %s: %v", channel, err)
			return false
		}
	}

	source = strings.TrimSpace(strings.ToLower(source))

	if prefixes, ok := disinformationSources[channel]; ok {
		for _, prefix := range prefixes {
			if strings.HasPrefix(source, prefix) {
				return true
			}
		}
	}

	return false
}

func (fs *Firestore) ReloadDisinformationSources(channel string) error {
	if len(disinformationSources) == 0 {
		disinformationSources = make(map[string][]string)
	}

	disinformationSources[channel] = make([]string, 0)

	disinformation, err := fs.DisinformationSources(channel)
	if err != nil {
		return err
	}

	for _, d := range disinformation {
		disinformationSources[channel] = append(disinformationSources[channel], d.Source)
	}

	return nil
}
