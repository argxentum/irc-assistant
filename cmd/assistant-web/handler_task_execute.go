package main

import (
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"io"
	"net/http"
	"strings"
)

func (s *server) taskExecuteHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()

	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != s.cfg.CloudTasks.AuthToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Errorf(nil, "error reading task execute request body: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	task, err := models.DeserializeTask(body)
	if err != nil {
		logger.Errorf(nil, "error deserializing task: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	logger.Debugf(nil, "executing cloud task %s: %s", task.ID, task.Type)

	if err := queue.GetDefault().Publish(task); err != nil {
		logger.Errorf(nil, "error publishing task to queue: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
