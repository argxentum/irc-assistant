package main

import (
	"assistant/pkg/api/commands"
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
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
		assistant, err = fs.CreateAssistant()
		if err != nil {
			logger.Errorf(nil, "error creating assistant, %s", err)
			return
		}
		logger.Debugf(nil, "assistant created")
	}

	processTasks(ctx, cfg, irc)
}

func initializeChannel(ctx context.Context, cfg *config.Config, irc irc.IRC, channel string) {
	logger := log.Logger()

	if slices.Contains(cfg.IRC.PostConnect.AutoLeave, channel) {
		logger.Debugf(nil, "channel %s is in auto leave list, leaving...", channel)
		irc.Part(channel)
		return
	}

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

func initializeChannelUser(ctx context.Context, cfg *config.Config, irc irc.IRC, channel string, mask *irc.Mask) {
	reg := commands.LoadCommandRegistry(ctx, cfg, irc)
	if cmd := reg.Command(commands.SummaryCommandName).(*commands.SummaryCommand); cmd != nil {
		cmd.InitializeUserPause(channel, mask.Nick, 15*time.Second)
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

	specifiedUser, err := repository.GetUserByNick(nil, channel, mask.Nick, false)
	if err != nil {
		logger.Errorf(nil, "error retrieving user, %s", err)
		return
	}

	if specifiedUser != nil {
		specifiedUser.UserID = mask.UserID
		specifiedUser.Host = mask.Host
		specifiedUser.UpdatedAt = time.Now()
		specifiedUser.IsAutoVoiced = specifiedUser.IsAutoVoiced || slices.Contains(ch.AutoVoiced, mask.Nick)

		if specifiedUser.IsAutoVoiced {
			irc.Voice(channel, mask.Nick)
		}

		if err = fs.UpdateUser(channel, specifiedUser, map[string]any{"is_auto_voiced": specifiedUser.IsAutoVoiced, "user_id": specifiedUser.UserID, "host": specifiedUser.Host, "updated_at": specifiedUser.UpdatedAt}); err != nil {
			panic(fmt.Errorf("error updating user, %s", err))
		}

		return
	}

	users, err := repository.GetUsersByHost(nil, channel, mask.Host)
	if err != nil {
		logger.Errorf(nil, "error getting users by host: %v", err)
		return
	}

	if len(users) == 0 {
		logger.Debugf(nil, "user %s not found, creating", mask.Nick)

		u := models.NewUser(mask)
		u.IsAutoVoiced = slices.Contains(ch.AutoVoiced, mask.Nick)
		err = fs.CreateUser(channel, u)
		if err != nil {
			panic(fmt.Errorf("error creating user, %s", err))
		}

		if len(ch.IntroMessages) > 0 {
			irc.SendMessages(mask.Nick, ch.IntroMessages)
		}

		if u.IsAutoVoiced {
			irc.Voice(channel, mask.Nick)
		}

		return
	}

	var alternateUser *models.User
	for _, u := range users {
		if specifiedUser != nil && alternateUser != nil {
			break
		}
		if u.Nick == mask.Nick {
			specifiedUser = u
		} else {
			alternateUser = u
		}
	}

	isAutoVoiced := slices.Contains(ch.AutoVoiced, mask.Nick)
	if alternateUser != nil {
		isAutoVoiced = isAutoVoiced || alternateUser.IsAutoVoiced
	}

	specifiedUser = models.NewUser(mask)
	specifiedUser.IsAutoVoiced = isAutoVoiced
	specifiedUser.CreatedAt = time.Now()
	specifiedUser.UpdatedAt = time.Now()

	if err = fs.CreateUser(channel, specifiedUser); err != nil {
		panic(fmt.Errorf("error creating user, %s", err))
	}

	if specifiedUser.IsAutoVoiced {
		irc.Voice(channel, mask.Nick)
	}
}
