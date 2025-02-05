package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/repository"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"os"
	"strings"
)

func main() {
	panic("Remove panic in main to load again...")

	ctx := context.NewContext()

	configFilename := ""
	if len(os.Args) > 1 {
		configFilename = os.Args[1]
	}

	if len(configFilename) == 0 {
		panic("config filename is required")
	}

	cfg, err := config.ReadConfig(configFilename)
	if err != nil {
		panic(err)
	}

	initializeFirestore(ctx, cfg)
	defer firestore.Get().Close()

	initializeLogger(ctx, cfg)
	defer log.Logger().Close()

	moveChannelBiasesToSources(cfg)
}

func initializeFirestore(ctx context.Context, cfg *config.Config) {
	if _, err := firestore.Initialize(ctx, cfg); err != nil {
		panic(fmt.Errorf("error initializing firestore, %s", err))
	}
}

func initializeLogger(ctx context.Context, cfg *config.Config) {
	if _, err := log.InitializeGCPLogger(ctx, cfg, "antifa-source-loader"); err != nil {
		panic(fmt.Errorf("error initializing GCP logger, %s", err))
	}
}

func moveChannelBiasesToSources(cfg *config.Config) {
	fs := firestore.Get()

	asst, err := repository.GetAssistant(nil, false)
	if err != nil {
		panic(err)
	}

	if asst == nil {
		panic("assistant not found")
	}

	for k, br := range asst.Cache.BiasResults {
		fmt.Printf("moving source: %s\n", k)

		br.Title = strings.TrimSpace(strings.Replace(br.Title, "–", "", 1))
		br.Rating = strings.TrimSpace(strings.ToLower(br.Rating))
		br.Factual = strings.TrimSpace(strings.ToLower(br.Factual))
		br.Credibility = strings.TrimSpace(strings.Replace(strings.ToLower(br.Credibility), "credibility", "", 1))

		keywords := strings.Split(strings.ToLower(k), " ")

		source := models.NewSource(br.Title, br.Rating, br.Factual, br.Credibility, br.DetailURL, []string{}, keywords)
		if err = fs.CreateSource(source); err != nil {
			panic(err)
		}
	}
}
