package main

import (
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"net/http"
	"strings"
)

func (s *server) shortcutHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()
	id := r.PathValue("id")

	// if id has an extension, strip out everything after the first dot
	if strings.Contains(id, ".") {
		parts := strings.SplitN(id, ".", 2)
		if len(parts) > 0 {
			id = parts[0]
		}
	}

	fs := firestore.Get()
	sc, err := fs.Shortcut(id)
	if err != nil {
		logger.Rawf(log.Error, "error searching for shortcut %s, %s", id, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	http.Redirect(w, r, sc.RedirectURL, http.StatusPermanentRedirect)
}
