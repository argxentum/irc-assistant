package queue

import (
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"cloud.google.com/go/pubsub"
	"context"
	"fmt"
	"google.golang.org/api/option"
)

var instance Queue

type Queue interface {
	Publish(task *models.Task) error
	Receive(callback func(*models.Task)) error
	Close() error
}

func Get() Queue {
	if instance == nil {
		panic("queue is not initialized")
	}

	return instance
}

func Initialize(ctx context.Context, cfg *config.Config) (Queue, error) {
	if instance != nil {
		return instance, nil
	}

	var client *pubsub.Client
	var err error
	client, err = pubsub.NewClient(ctx, cfg.GoogleCloud.ProjectID, option.WithCredentialsFile(cfg.GoogleCloud.ServiceAccountFilename))
	if err != nil {
		return nil, fmt.Errorf("error creating firestore client, %s", err)
	}

	topic := client.Topic(cfg.Queue.Topic)
	if topic == nil {
		return nil, fmt.Errorf("invalid topic, %s", cfg.Queue.Topic)
	}

	subscription := client.Subscription(cfg.Queue.Subscription)
	if subscription == nil {
		return nil, fmt.Errorf("invalid subscription, %s", cfg.Queue.Subscription)
	}

	instance = &queue{
		ctx:          ctx,
		cfg:          cfg,
		client:       client,
		topic:        topic,
		subscription: subscription,
	}

	return instance, nil
}

type queue struct {
	ctx          context.Context
	cfg          *config.Config
	client       *pubsub.Client
	topic        *pubsub.Topic
	subscription *pubsub.Subscription
}

func (q *queue) Close() error {
	q.topic.Stop()
	return q.client.Close()
}

func (q *queue) Publish(task *models.Task) error {
	logger := log.Logger()

	data, err := task.Serialize()
	if err != nil {
		return fmt.Errorf("error serializing task, %s", err)
	}

	_ = q.topic.Publish(q.ctx, &pubsub.Message{
		Data: data,
	})

	logger.Debugf(nil, "published: %s", string(data))

	return nil
}

func (q *queue) Receive(callback func(*models.Task)) error {
	logger := log.Logger()

	return q.subscription.Receive(q.ctx, func(ctx context.Context, msg *pubsub.Message) {
		logger.Debugf(nil, "received: %s", string(msg.Data))

		task, err := models.DeserializeTask(msg.Data)
		if err != nil {
			logger.Errorf(nil, "error deserializing task, %s", err)
			return
		}

		msg.Ack()
		callback(task)
	})
}
