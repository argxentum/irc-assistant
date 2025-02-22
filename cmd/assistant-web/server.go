package main

import (
	"assistant/pkg/config"
	"assistant/pkg/log"
	"context"
	"fmt"
	nativeLog "log"
	"net/http"
)

type server struct {
	ctx context.Context
	cfg *config.Config
}

var templatesRoot = "cmd/assistant-web/templates"

func (s *server) start() {
	logger := log.Logger()
	logger.Rawf(log.Info, "starting %s on :%d", s.cfg.Web.ExternalRootURL, s.cfg.Web.Port)

	// misc routes
	http.HandleFunc("/", s.defaultHandler)
	http.HandleFunc("/text/{text}", s.giphyAnimatedTextHandler)
	http.HandleFunc("/animated/{text}", s.giphyAnimatedTextHandler)
	http.HandleFunc("/gifs/{q}", s.giphySearchHandler)
	http.HandleFunc("/gif/{q}", s.giphySearchHandler)

	// page routes
	http.HandleFunc("/about", s.aboutPageHandler)

	nativeLog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Web.Port), nil))
}
