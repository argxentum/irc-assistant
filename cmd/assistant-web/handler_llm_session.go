package main

import (
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type sessionEntry struct {
	ID       string
	Prompt   template.JS
	Content  template.JS
	Created  template.JS
	Complete bool
}

type sessionPollEntry struct {
	ID       string `json:"id"`
	Prompt   string `json:"prompt"`
	Content  string `json:"content"`
	Created  string `json:"created"`
	Complete bool   `json:"complete"`
}

type sessionPollResponse struct {
	Entries       []sessionPollEntry `json:"entries"`
	AnyProcessing bool               `json:"anyProcessing"`
}

func (s *server) llmSessionPollHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()
	id := r.PathValue("id")

	fs := firestore.Get()
	responses, err := fs.LLMResponsesBySession(id)
	if err != nil {
		logger.Rawf(log.Error, "error fetching session %s, %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	poll := sessionPollResponse{Entries: make([]sessionPollEntry, 0, len(responses))}
	for _, resp := range responses {
		poll.Entries = append(poll.Entries, sessionPollEntry{
			ID:       resp.ID,
			Prompt:   resp.Prompt,
			Content:  resp.Content,
			Created:  resp.CreatedAt.UTC().Format(time.RFC3339),
			Complete: resp.Complete,
		})
		if !resp.Complete {
			poll.AnyProcessing = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(poll)
}

func (s *server) llmSessionHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()
	id := r.PathValue("id")

	fs := firestore.Get()
	responses, err := fs.LLMResponsesBySession(id)
	if err != nil {
		logger.Rawf(log.Error, "error fetching session %s, %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(responses) == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	t, err := template.ParseFiles(templatesRoot + "/llm_session.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	entries := make([]sessionEntry, 0, len(responses))
	anyProcessing := false
	for _, resp := range responses {
		promptJSON, _ := json.Marshal(resp.Prompt)
		contentJSON, _ := json.Marshal(resp.Content)
		createdJSON, _ := json.Marshal(resp.CreatedAt.UTC().Format(time.RFC3339))
		entries = append(entries, sessionEntry{
			ID:       resp.ID,
			Prompt:   template.JS(promptJSON),
			Content:  template.JS(contentJSON),
			Created:  template.JS(createdJSON),
			Complete: resp.Complete,
		})
		if !resp.Complete {
			anyProcessing = true
		}
	}

	first := responses[0]
	args := map[string]any{
		"name":          s.cfg.IRC.Nick,
		"channel":       first.Channel,
		"nick":          first.Nick,
		"entries":       entries,
		"anyProcessing": anyProcessing,
	}

	if err = t.Execute(w, args); err != nil {
		http.Error(w, fmt.Sprintf("error executing template: %v", err), http.StatusInternalServerError)
	}
}
