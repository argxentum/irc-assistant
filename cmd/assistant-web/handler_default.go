package main

import "net/http"

func (s *server) defaultHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, s.cfg.Web.DefaultRedirect, http.StatusSeeOther)
}
