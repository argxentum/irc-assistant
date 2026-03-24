package queue

import (
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

const defaultQueue = "default"
const proxyQueue = "proxy"

var instances = map[string]Queue{}

type Queue interface {
	Publish(task *models.Task) error
	Receive(callback func(*models.Task)) error
	Clear() error
	Close() error
}

func GetDefault() Queue {
	return getNamed(defaultQueue)
}

func GetProxy() Queue {
	return getNamed(proxyQueue)
}

func getNamed(name string) Queue {
	q, ok := instances[name]
	if !ok {
		panic(fmt.Sprintf("queue %s is not initialized", name))
	}
	return q
}

func InitializeDefault(ctx context.Context, cfg *config.Config, topic, subscription string) (Queue, error) {
	return initializeNamed(ctx, cfg, defaultQueue, topic, subscription)
}

func InitializeProxy(ctx context.Context, cfg *config.Config, topic, subscription string) (Queue, error) {
	return initializeNamed(ctx, cfg, proxyQueue, topic, subscription)
}

func initializeNamed(ctx context.Context, cfg *config.Config, name, topic, subscription string) (Queue, error) {
	if q, ok := instances[name]; ok {
		return q, nil
	}

	client, err := pubsub.NewClient(ctx, cfg.GoogleCloud.ProjectID, option.WithCredentialsFile(cfg.GoogleCloud.ServiceAccountFilename))
	if err != nil {
		return nil, fmt.Errorf("error creating pubsub client, %s", err)
	}

	t := client.Topic(topic)
	if t == nil {
		return nil, fmt.Errorf("invalid topic, %s", topic)
	}

	var s *pubsub.Subscription
	if subscription != "" {
		s = client.Subscription(subscription)
		if s == nil {
			return nil, fmt.Errorf("invalid subscription, %s", subscription)
		}
	}

	q := &queue{
		ctx:          ctx,
		cfg:          cfg,
		client:       client,
		topic:        t,
		subscription: s,
	}

	instances[name] = q
	return q, nil
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
	if q.subscription == nil {
		return fmt.Errorf("queue has no subscription configured")
	}

	logger := log.Logger()

	return q.subscription.Receive(q.ctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()
		logger.Debugf(nil, "received: %s", string(msg.Data))

		task, err := models.DeserializeTask(msg.Data)
		if err != nil {
			logger.Errorf(nil, "error deserializing task, %s", err)
			return
		}

		callback(task)
	})
}

func (q *queue) Clear() error {
	if q.subscription == nil {
		return nil
	}
	return q.subscription.SeekToTime(q.ctx, time.Now())
}
