package firestore

import (
	"assistant/pkg/models"
	"fmt"

	"cloud.google.com/go/firestore"
)

func (fs *Firestore) GetSource(id string) (*models.Source, error) {
	return get[models.Source](fs.ctx, fs.client, fs.pathToSource(id))
}

func (fs *Firestore) SetSource(source *models.Source) error {
	return set(fs.ctx, fs.client, fs.pathToSource(source.ID), source)
}

func (fs *Firestore) UpdateSource(id string, fields map[string]any) error {
	return update(fs.ctx, fs.client, fs.pathToSource(id), fields)
}

func (fs *Firestore) CreateSource(source *models.Source) error {
	return create(fs.ctx, fs.client, fs.pathToSource(source.ID), source)
}

func (fs *Firestore) DeleteSource(id string) error {
	return remove(fs.ctx, fs.client, fs.pathToSource(id))
}

func (fs *Firestore) FindSourcesByDomain(input string) ([]*models.Source, error) {
	criteria := QueryCriteria{
		Path: fs.pathToSources(),
		Filter: firestore.PropertyFilter{
			Path:     "urls",
			Operator: ArrayContains,
			Value:    input,
		},
	}

	return query[models.Source](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) FindSourcesByKeywords(input []string) ([]*models.Source, error) {
	criteria := QueryCriteria{
		Path: fs.pathToSources(),
		Filter: firestore.PropertyFilter{
			Path:     "keywords",
			Operator: ArrayContainsAny,
			Value:    input,
		},
	}

	return query[models.Source](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) pathToSources() string {
	return fmt.Sprintf("%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathSources)
}

func (fs *Firestore) pathToSource(id string) string {
	return fmt.Sprintf("%s/%s", fs.pathToSources(), id)
}
