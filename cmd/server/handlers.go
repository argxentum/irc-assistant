package main

import (
	"assistant/pkg/api/giphy"
	"assistant/pkg/log"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
)

func (s *server) defaultHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (s *server) animatedTextHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()

	animatedText, err := giphy.GetAnimatedText(s.cfg, r.PathValue("text"))
	if err != nil {
		logger.Rawf(log.Error, "error getting giphy animated text, %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := http.Get(animatedText.Data[rand.IntN(len(animatedText.Data))].URL)
	if err != nil {
		logger.Rawf(log.Error, "error fetching giphy animated text data, %s", err)
		http.Error(w, "generation error", http.StatusInternalServerError)
		return
	}

	if resp == nil {
		logger.Rawf(log.Error, "no response from giphy for animated text")
		http.Error(w, "retrieval error", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		logger.Rawf(log.Error, "error getting giphy animated text data, status %s", resp.Status)
		http.Error(w, "retrieval status error", http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	w.Header().Set("Content-Type", "image/gif")

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Rawf(log.Error, "error reading giphy animated text data, %s", err)
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("Cache-Control", "no-cache")

	if _, err := w.Write(data); err != nil {
		logger.Rawf(log.Error, "error writing giphy animated text data, %s", err)
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}
}
