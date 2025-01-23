package main

import (
	"assistant/pkg/api/commands"
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
	"slices"
	"time"
)

func initializeLogger(ctx context.Context, cfg *config.Config) {
	_, err := log.InitializeGCPLogger(ctx, cfg, cfg.IRC.Nick)
	if err != nil {
		panic(fmt.Errorf("error initializing logger, %s", err))
	}
}

func initializeFirestore(ctx context.Context, cfg *config.Config) {
	_, err := firestore.Initialize(ctx, cfg)
	if err != nil {
		panic(fmt.Errorf("error initializing firestore, %s", err))
	}
}

func initializeQueue(ctx context.Context, cfg *config.Config) {
	q, err := queue.Initialize(ctx, cfg)
	if err != nil {
		panic(fmt.Errorf("error initializing queue, %s", err))
	}

	err = q.Clear()
	if err != nil {
		panic(fmt.Errorf("error clearing queue, %s", err))
	}
}

func initializeAssistant(ctx context.Context, cfg *config.Config, irc irc.IRC) {
	logger := log.Logger()
	fs := firestore.Get()

	assistant, err := fs.Assistant()
	if err != nil {
		logger.Errorf(nil, "error retrieving assistant, %s", err)
		return
	}

	if assistant == nil {
		logger.Debugf(nil, "assistant not found, creating")
		err = fs.CreateAssistant()
		if err != nil {
			logger.Errorf(nil, "error creating assistant, %s", err)
			return
		}
		logger.Debugf(nil, "assistant created")
	}

	processTasks(ctx, cfg, irc)
}

func initializeChannel(ctx context.Context, cfg *config.Config, channel string) {
	logger := log.Logger()
	fs := firestore.Get()
	logger.Rawf(log.Debug, "loading banned words for channel %s", channel)

	ch, err := fs.Channel(channel)
	if err != nil {
		panic(fmt.Errorf("error retrieving channel, %s", err))
	}

	if ch == nil {
		logger.Debugf(nil, "channel %s not found, creating", channel)
		err = fs.CreateChannel(models.NewChannel(channel, cfg.IRC.Inactivity.DefaultDuration))
		if err != nil {
			panic(fmt.Errorf("error creating channel, %s", err))
		}
		logger.Debugf(nil, "channel %s created", channel)
	}

	bannedWords, err := fs.BannedWords(channel)
	if err != nil {
		panic(fmt.Errorf("error retrieving banned words, %s", err))
	}

	for _, word := range bannedWords {
		ctx.Session().AddBannedWord(channel, word.Word)
	}

	logger.Rawf(log.Debug, "loaded %d banned words for channel %s", len(bannedWords), channel)

	path := fs.PersistentChannelTaskPath(channel, models.ChannelInactivityTaskID)
	task, err := fs.Task(path)
	if err != nil {
		panic(fmt.Errorf("error retrieving persistent task, %s", err))
	}

	if task == nil {
		logger.Debugf(nil, "channel %s inactivity persistent task not found, creating", channel)
		duration, err := elapse.ParseDuration(cfg.IRC.Inactivity.DefaultDuration)
		if err != nil {
			logger.Errorf(nil, "error parsing default inactivity duration, %s", err)
		}
		err = fs.SetPersistentChannelTaskDue(channel, models.ChannelInactivityTaskID, duration)
		if err != nil {
			panic(fmt.Errorf("error creating persistent task, %s", err))
		}
		logger.Debugf(nil, "channel %s inactivity persistent task created", channel)
	}
}

func initializeChannelUser(ctx context.Context, cfg *config.Config, irc irc.IRC, channel, nick string) {
	reg := commands.LoadCommandRegistry(ctx, cfg, irc)
	if cmd := reg.Command(commands.SummaryCommandName).(*commands.SummaryCommand); cmd != nil {
		cmd.InitializeUserRateLimit(channel, nick, 15*time.Second)
	}

	logger := log.Logger()
	fs := firestore.Get()

	ch, err := fs.Channel(channel)
	if err != nil {
		panic(fmt.Errorf("error retrieving channel, %s", err))
	}

	if ch == nil {
		logger.Debugf(nil, "channel %s not found, creating", channel)
		ch = models.NewChannel(channel, cfg.IRC.Inactivity.DefaultDuration)
		err = fs.CreateChannel(ch)
		if err != nil {
			panic(fmt.Errorf("error creating channel, %s", err))
		}
	}

	if slices.Contains(ch.AutoVoiced, nick) {
		irc.Voice(channel, nick)
	}
}
