package main

import (
	"html/template"
	"net/http"
)

func (s *server) rulesPageHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("templates/rules.html")
	_ = t.Execute(w, nil)
}
