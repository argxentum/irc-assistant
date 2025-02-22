package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func (s *server) aboutPageHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles(templatesRoot + "/about.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	args := map[string]any{
		"name": s.cfg.IRC.Nick,
		"url":  s.cfg.Web.ExternalRootURL,
	}

	err = t.Execute(w, args)
	if err != nil {
		http.Error(w, fmt.Sprintf("error executing template: %v", err), http.StatusInternalServerError)
	}
}
