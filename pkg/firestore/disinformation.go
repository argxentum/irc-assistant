package firestore

import (
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
)

var disinformationSources map[string][]string

func (fs *Firestore) pathToDisinformation(channel string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathDisinformation)
}

func (fs *Firestore) Disinformation(channel string) ([]*models.Disinformation, error) {
	return list[models.Disinformation](fs.ctx, fs.client, fs.pathToDisinformation(channel))
}

func (fs *Firestore) AddDisinformation(channel, source string) error {
	logger := log.Logger()
	if fs.IsDisinformation(channel, source) {
		logger.Debugf(nil, "disinformation already exists: %s", source)
		return nil
	}
	return create(fs.ctx, fs.client, fs.pathToDisinformation(channel), models.NewDisinformation(source))
}

func (fs *Firestore) IsDisinformation(channel, url string) bool {
	if disinformationSources == nil {
		disinformationSources = make(map[string][]string)
	}

	if sources, ok := disinformationSources[channel]; ok {
		for _, source := range sources {
			if source == url {
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

	disinformation, err := fs.Disinformation(channel)
	if err != nil {
		return err
	}

	for _, d := range disinformation {
		disinformationSources[channel] = append(disinformationSources[channel], d.Source)
	}

	return nil
}
