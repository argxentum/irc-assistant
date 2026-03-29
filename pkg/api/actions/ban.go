package actions

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"time"
)

func Ban(ircs irc.IRC, channel, mask, duration, reason string) {
	logger := log.Logger()

	// build kick reason with duration info, if duration specified
	if duration != "" {
		durationDesc := elapse.ParseDurationDescription(duration)
		if reason == "" {
			reason = fmt.Sprintf("temporarily banned for %s", durationDesc)
		} else {
			reason = fmt.Sprintf("%s - temporarily banned for %s", reason, durationDesc)
		}
	}

	// kick all users matching the ban mask
	done := make(chan []*irc.User, 1)
	ircs.ListUsersByMask(channel, mask, func(users []*irc.User) {
		done <- users
	})
	matchedUsers := <-done
	time.Sleep(250 * time.Millisecond)
	for _, u := range matchedUsers {
		ircs.Kick(channel, u.Mask.Nick, reason)
		logger.Infof(nil, "ban: kicked %s from %s: %s", u.Mask.Nick, channel, reason)
		time.Sleep(25 * time.Millisecond)
	}

	// set the ban
	ircs.Ban(channel, mask)
	logger.Infof(nil, "ban: banned %s in %s", mask, channel)

	// schedule ban removal if duration specified
	if duration != "" {
		dur, err := elapse.ParseDuration(duration)
		if err != nil {
			logger.Errorf(nil, "ban: error parsing duration: %s", err)
			return
		}

		m := irc.ParseMask(mask)
		task := models.NewBanRemovalTask(time.Now().Add(dur), m.String(), channel)
		if err := firestore.Get().AddTask(task); err != nil {
			logger.Errorf(nil, "ban: error scheduling ban removal: %s", err)
		}
	}
}
