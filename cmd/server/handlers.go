package main

import (
	"assistant/pkg/api/giphy"
	"assistant/pkg/log"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
)

func (s *server) defaultHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, s.cfg.Server.DefaultRedirect, http.StatusSeeOther)
}

func (s *server) giphyAnimatedTextHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()

	text := r.PathValue("text")
	text = strings.Replace(text, "_", " ", -1)
	text = strings.TrimSuffix(text, ".gif")

	animatedText, err := giphy.CreateAnimatedText(s.cfg, text)
	if err != nil {
		logger.Rawf(log.Error, "error getting giphy animated text, %s", err)
		http.Error(w, "request error", http.StatusInternalServerError)
		return
	}

	if len(animatedText.Data) == 0 {
		logger.Rawf(log.Error, "no data returned for giphy animated text")
		http.Error(w, "no data", http.StatusNotFound)
	}

	url := animatedText.Data[rand.IntN(len(animatedText.Data))].Images["original"].URL
	if len(url) == 0 {
		logger.Rawf(log.Error, "no url returned for giphy animated text")
		http.Error(w, "bad url", http.StatusNotFound)
		return
	}

	resp, err := http.Get(url)
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Rawf(log.Error, "error reading giphy animated text data, %s", err)
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Pragma", "no-cache")

	if _, err = w.Write(data); err != nil {
		logger.Rawf(log.Error, "error writing giphy animated text, %s", err)
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}
}

func (s *server) giphySearchHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()

	query := r.PathValue("q")
	query = strings.Replace(query, "_", " ", -1)
	query = strings.TrimSuffix(query, ".gif")

	gif, err := giphy.SearchGIFs(s.cfg, query)
	if err != nil {
		logger.Rawf(log.Error, "error searching for gif, %s", err)
		http.Error(w, "request error", http.StatusInternalServerError)
		return
	}

	if len(gif.Data) == 0 {
		logger.Rawf(log.Error, "no data returned for giphy gif search")
		http.Error(w, "no data", http.StatusNotFound)
	}

	url := gif.Data[0].Images["original"].URL
	if len(url) == 0 {
		logger.Rawf(log.Error, "no url returned for giphy animated text")
		http.Error(w, "bad url", http.StatusNotFound)
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		logger.Rawf(log.Error, "error fetching searched gif data, %s", err)
		http.Error(w, "generation error", http.StatusInternalServerError)
		return
	}

	if resp == nil {
		logger.Rawf(log.Error, "no response from giphy gif query")
		http.Error(w, "retrieval error", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		logger.Rawf(log.Error, "error getting gif data, status %s", resp.Status)
		http.Error(w, "retrieval status error", http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Rawf(log.Error, "error reading giphy animated query data, %s", err)
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("Cache-Control", "public, max-age=86400")

	if _, err = w.Write(data); err != nil {
		logger.Rawf(log.Error, "error writing giphy gif, %s", err)
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}
}
