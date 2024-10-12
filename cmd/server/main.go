package main

import (
	"assistant/pkg/config"
	"fmt"
	"net/http"
	"os"
)

const defaultConfigFilename = "config.yaml"

func main() {
	configFilename := defaultConfigFilename
	if len(os.Args) > 1 {
		configFilename = os.Args[1]
	}

	cfg, err := config.ReadConfig(configFilename)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), nil)
}
