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

func (s *server) llmResponseHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()
	id := r.PathValue("id")

	fs := firestore.Get()
	resp, err := fs.LLMResponse(id)
	if err != nil {
		logger.Rawf(log.Error, "error fetching LLM response %s, %s", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	t, err := template.ParseFiles(templatesRoot + "/llm_response.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	contentJSON, err := json.Marshal(resp.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("error encoding content: %v", err), http.StatusInternalServerError)
		return
	}

	createdJSON, err := json.Marshal(resp.CreatedAt.UTC().Format(time.RFC3339))
	if err != nil {
		http.Error(w, fmt.Sprintf("error encoding timestamp: %v", err), http.StatusInternalServerError)
		return
	}

	args := map[string]any{
		"name":    s.cfg.IRC.Nick,
		"id":      resp.ID,
		"channel": resp.Channel,
		"nick":    resp.Nick,
		"prompt":  resp.Prompt,
		"content": template.JS(contentJSON),
		"created": template.JS(createdJSON),
	}

	if err = t.Execute(w, args); err != nil {
		http.Error(w, fmt.Sprintf("error executing template: %v", err), http.StatusInternalServerError)
	}
}
