package firestore

import (
	"assistant/pkg/config"
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"google.golang.org/api/option"
)

var instance *Firestore

type Firestore struct {
	ctx    context.Context
	cfg    *config.Config
	client *firestore.Client
}

func Get() *Firestore {
	if instance == nil {
		panic("firestore is not initialized")
	}

	return instance
}

func Initialize(ctx context.Context, cfg *config.Config) (*Firestore, error) {
	if instance != nil {
		return instance, nil
	}

	var client *firestore.Client
	var err error
	client, err = firestore.NewClient(ctx, cfg.GoogleCloud.ProjectID, option.WithCredentialsFile(cfg.GoogleCloud.ServiceAccountFilename))
	if err != nil {
		return nil, fmt.Errorf("error creating firestore client, %s", err)
	}

	instance = &Firestore{
		ctx:    ctx,
		cfg:    cfg,
		client: client,
	}

	return instance, nil
}

func (fs *Firestore) Close() error {
	return fs.client.Close()
}
