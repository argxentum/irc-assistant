package main

import (
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	nativeLog "log"
	"net/http"
)

type server struct {
	cfg *config.Config
}

func (s *server) start() {
	logger := log.Logger()
	logger.Rawf(log.Info, "starting %s on :%d", s.cfg.Server.ExternalRootURL, s.cfg.Server.Port)

	http.HandleFunc("/", s.defaultHandler)
	http.HandleFunc("/animated/{text}", s.animatedTextHandler)
	nativeLog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Server.Port), nil))
}
