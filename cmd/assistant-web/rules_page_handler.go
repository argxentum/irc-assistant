package main

import (
	"html/template"
	"net/http"
)

func (s *server) rulesPageHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("/templates/rules.html")
	if err != nil {
		http.Error(w, "error parsing template", http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, nil)
	if err != nil {
		http.Error(w, "error executing template", http.StatusInternalServerError)
	}
}
