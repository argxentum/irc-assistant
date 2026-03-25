package main

import (
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
	"sync"
)

type proxy struct {
	cfg      *config.Config
	sessions map[string]*session
	mu       sync.Mutex
}

func (p *proxy) start() {
	logger := log.Logger()
	logger.Debug(nil, "starting proxy")

	err := queue.GetProxy().Receive(func(task *models.Task) {
		logger.Debugf(nil, "received task %s: %s", task.ID, task.Type)

		var err error
		switch task.Type {
		case models.TaskTypeProxyLLMRequest:
			err = p.handleLLMProxyRequest(task)
		case models.TaskTypeProxySummaryRequest:
			err = p.handleSummaryProxyRequest(task)
		case models.TaskTypeProxyInactivityRequest:
			err = p.handleInactivityProxyRequest(task)
		default:
			logger.Warningf(nil, "unknown task type: %s", task.Type)
		}

		if err != nil {
			logger.Errorf(nil, "error handling task %s: %s", task.ID, err)
		}
	})

	if err != nil {
		logger.Errorf(nil, "error receiving tasks: %s", err)
	}
}

func (p *proxy) handleLLMProxyRequest(task *models.Task) error {
	data := task.Data.(models.ProxyLLMRequestTaskData)
	logger := log.Logger()
	logger.Debugf(nil, "handling proxy request %s [%s] in %s from %s", task.ID, data.Handler, data.Channel, data.Nick)

	switch data.Handler {
	case "llm":
		return p.handleLLM(task.ID, data)
	default:
		return fmt.Errorf("unknown handler: %s", data.Handler)
	}
}

func (p *proxy) publishResponse(requestID, channel, nick, responseID, sessionID string, processing bool) error {
	task := models.NewProxyLLMResponseTask(requestID, channel, nick, responseID, sessionID, processing)
	return queue.GetDefault().Publish(task)
}
