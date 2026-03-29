package firestore

import (
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
)

func (fs *Firestore) ListSources() ([]*models.Source, error) {
	cr := fs.client.Collection(fs.pathToSources())
	if cr == nil {
		return nil, fmt.Errorf("invalid collection path, %s", fs.pathToSources())
	}

	iter := cr.Documents(fs.ctx)
	ds, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error listing sources: %w", err)
	}

	sources := make([]*models.Source, 0, len(ds))
	for _, d := range ds {
		data := d.Data()
		src := &models.Source{
			ID:          stringVal(data, "id"),
			Title:       stringVal(data, "title"),
			Bias:        stringVal(data, "bias"),
			Factuality:  stringVal(data, "factuality"),
			Credibility: stringVal(data, "credibility"),
			Reviews:     stringSliceVal(data, "reviews"),
			URLs:        stringSliceVal(data, "urls"),
			Paywall:     boolVal(data, "paywall"),
			Keywords:    stringSliceVal(data, "keywords"),
			Citations:   intVal(data, "citations"),
			CreatedAt:   timeVal(data, "created_at"),
			UpdatedAt:   timeVal(data, "updated_at"),
		}
		sources = append(sources, src)
	}

	return sources, nil
}

func stringVal(data map[string]any, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func intVal(data map[string]any, key string) int {
	switch v := data[key].(type) {
	case int64:
		return int(v)
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func boolVal(data map[string]any, key string) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return false
}

func timeVal(data map[string]any, key string) time.Time {
	if v, ok := data[key].(time.Time); ok {
		return v
	}
	return time.Time{}
}

func stringSliceVal(data map[string]any, key string) []string {
	switch v := data[key].(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		if v != "" {
			log.Logger().Debugf(nil, "source field %s is string instead of array, converting", key)
			return []string{v}
		}
		return []string{}
	default:
		return []string{}
	}
}

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

func (fs *Firestore) IncrementSourceCitations(id string) error {
	return update(fs.ctx, fs.client, fs.pathToSource(id), map[string]any{
		"citations":  firestore.Increment(1),
		"updated_at": time.Now(),
	})
}

func (fs *Firestore) IncrementUnknownSource(domain string) error {
	path := fs.pathToUnknownSource(domain)
	doc := fs.client.Doc(path)
	_, err := doc.Set(fs.ctx, map[string]any{
		"domain":     domain,
		"citations":  firestore.Increment(1),
		"updated_at": time.Now(),
	}, firestore.MergeAll)
	return err
}

func (fs *Firestore) ListUnknownSources() ([]*models.UnknownSource, error) {
	return list[models.UnknownSource](fs.ctx, fs.client, fs.pathToUnknownSources())
}

func (fs *Firestore) DeleteUnknownSource(domain string) error {
	return remove(fs.ctx, fs.client, fs.pathToUnknownSource(domain))
}

func (fs *Firestore) pathToUnknownSources() string {
	return fmt.Sprintf("%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathUnknownSources)
}

func (fs *Firestore) pathToUnknownSource(domain string) string {
	return fmt.Sprintf("%s/%s", fs.pathToUnknownSources(), domain)
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
