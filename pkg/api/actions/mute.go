package actions

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"slices"
	"time"
)

func Mute(ircs irc.IRC, channel, nick, host, duration, reason string) {
	logger := log.Logger()
	fs := firestore.Get()

	// send channel notification
	var msg string
	if duration != "" {
		msg = fmt.Sprintf("\U0001F507 Temporarily muting %s for %s", style.Bold(nick), style.Bold(elapse.ParseDurationDescription(duration)))
	} else {
		msg = fmt.Sprintf("\U0001F507 Muting %s", style.Bold(nick))
	}
	if reason != "" {
		msg += ": " + reason
	}
	ircs.SendMessage(channel, msg)

	// collect all users sharing the same host
	users := make([]*models.User, 0)
	if host != "" {
		hostUsers, err := fs.GetUsersByHost(channel, host)
		if err != nil {
			logger.Errorf(nil, "mute: error getting users by host: %s", err)
		} else {
			users = hostUsers
		}
	}

	// ensure the target nick is included
	hasTarget := false
	for _, u := range users {
		if u.Nick == nick {
			hasTarget = true
			break
		}
	}
	if !hasTarget {
		users = append([]*models.User{{Nick: nick}}, users...)
	}

	// check auto-voice status before removing it
	ch, err := fs.Channel(channel)
	if err != nil {
		logger.Errorf(nil, "mute: error getting channel: %s", err)
	}

	isAutoVoiced := false
	if ch != nil {
		isAutoVoiced = slices.Contains(ch.AutoVoiced, nick)
	}
	for _, u := range users {
		if u.IsAutoVoiced {
			isAutoVoiced = true
			break
		}
	}

	// mute all users and remove auto-voice
	channelAutoVoiceChanged := false
	for _, u := range users {
		ircs.Mute(channel, u.Nick)
		if duration != "" {
			logger.Infof(nil, "mute: temporarily muted %s in %s for %s", u.Nick, channel, duration)
		} else {
			logger.Infof(nil, "mute: muted %s in %s", u.Nick, channel)
		}

		if ch != nil && slices.Contains(ch.AutoVoiced, u.Nick) {
			ch.AutoVoiced = slices.DeleteFunc(ch.AutoVoiced, func(n string) bool { return n == u.Nick })
			channelAutoVoiceChanged = true
			logger.Debugf(nil, "mute: removed channel auto-voice from %s", u.Nick)
		}

		if u.IsAutoVoiced {
			u.IsAutoVoiced = false
			if err := fs.UpdateUser(channel, u, map[string]any{"is_auto_voiced": false, "updated_at": time.Now()}); err != nil {
				logger.Errorf(nil, "mute: error updating user isAutoVoiced: %s", err)
			}
			logger.Debugf(nil, "mute: removed auto-voice from user %s", u.Nick)
		}
	}

	if channelAutoVoiceChanged {
		if err := fs.UpdateChannel(channel, map[string]any{"auto_voiced": ch.AutoVoiced, "updated_at": time.Now()}); err != nil {
			logger.Errorf(nil, "mute: error updating channel auto_voiced: %s", err)
		}
	}

	// schedule mute removal if duration specified
	if duration != "" {
		dur, err := elapse.ParseDuration(duration)
		if err != nil {
			logger.Errorf(nil, "mute: error parsing duration: %s", err)
			return
		}

		task := models.NewMuteRemovalTask(time.Now().Add(dur), channel, nick, host, isAutoVoiced)
		if err := fs.AddTask(task); err != nil {
			logger.Errorf(nil, "mute: error scheduling mute removal: %s", err)
		}
	}
}
