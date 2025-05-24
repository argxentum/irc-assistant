package main

import (
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"net/http"
)

func (s *server) shortcutHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()
	id := r.PathValue("id")

	fs := firestore.Get()
	sc, err := fs.Shortcut(id)
	if err != nil {
		logger.Rawf(log.Error, "error searching for shortcut %s, %s", id, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	http.Redirect(w, r, sc.RedirectURL, http.StatusPermanentRedirect)
}
