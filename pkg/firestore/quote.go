package firestore

import (
	"assistant/pkg/models"
	"cloud.google.com/go/firestore"
	"fmt"
)

func (fs *Firestore) Quotes(channel string) ([]*models.Quote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathQuotes)
	return list[models.Quote](fs.ctx, fs.client, path)
}

func (fs *Firestore) FindUserQuotes(channel, nick string) ([]*models.Quote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathQuotes)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.PropertyFilter{
			Path:     "author",
			Operator: Equal,
			Value:    nick,
		},
		OrderBy: []OrderBy{
			{
				Field:     "quoted_at",
				Direction: firestore.Desc,
			},
		},
	}

	return query[models.Quote](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) FindUserQuotesWithContent(channel, nick string, keywords []string) ([]*models.Quote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathQuotes)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.AndFilter{
			Filters: []firestore.EntityFilter{
				firestore.PropertyFilter{
					Path:     "author",
					Operator: Equal,
					Value:    nick,
				},
				firestore.PropertyFilter{
					Path:     "keywords",
					Operator: ArrayContainsAny,
					Value:    keywords,
				},
			},
		},
		OrderBy: []OrderBy{
			{
				Field:     "quoted_at",
				Direction: firestore.Desc,
			},
		},
	}

	return query[models.Quote](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) FindQuotes(channel string, keywords []string) ([]*models.Quote, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathQuotes)

	criteria := QueryCriteria{
		Path: path,
		Filter: firestore.PropertyFilter{
			Path:     "keywords",
			Operator: ArrayContainsAny,
			Value:    keywords,
		},
		OrderBy: []OrderBy{
			{
				Field:     "quoted_at",
				Direction: firestore.Desc,
			},
		},
	}

	return query[models.Quote](fs.ctx, fs.client, criteria)
}

func (fs *Firestore) CreateQuote(channel string, quote *models.Quote) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathQuotes, quote.ID)
	return create(fs.ctx, fs.client, path, quote)
}

func (fs *Firestore) UpdateQuote(channel string, quote *models.Quote, fields map[string]any) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s/%s", pathAssistants, fs.cfg.IRC.Nick, pathChannels, channel, pathQuotes, quote.ID)
	return update(fs.ctx, fs.client, path, fields)
}
