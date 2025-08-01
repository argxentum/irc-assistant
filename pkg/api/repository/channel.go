package repository

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"cmp"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

func GetAllChannels(e *irc.Event) ([]*models.Channel, error) {
	fs := firestore.Get()

	channels, err := fs.Channels()
	if err != nil {
		return nil, fmt.Errorf("error retrieving channels, %w", err)
	}

	if channels == nil {
		return nil, fmt.Errorf("no channels found")
	}

	for _, ch := range channels {
		populateChannelDefaults(ch)
	}

	return channels, nil
}

func GetChannel(e *irc.Event, channel string) (*models.Channel, error) {
	fs := firestore.Get()

	ch, err := fs.Channel(channel)
	if err != nil {
		return nil, fmt.Errorf("error retrieving channel, %w", err)
	}

	if ch == nil {
		return nil, fmt.Errorf("channel %s does not exist", channel)
	}

	populateChannelDefaults(ch)

	return ch, nil
}

func populateChannelDefaults(ch *models.Channel) {
	if ch.AutoVoiced == nil {
		ch.AutoVoiced = make([]string, 0)
	}

	if ch.VoiceRequests == nil {
		ch.VoiceRequests = make([]models.VoiceRequest, 0)
	}

	slices.SortFunc(ch.VoiceRequests, func(a, b models.VoiceRequest) int {
		return cmp.Compare(a.RequestedAt.Unix(), b.RequestedAt.Unix())
	})
}

func IsChannelAutoVoicedUser(e *irc.Event, ch *models.Channel, nick string) bool {
	return slices.Contains(ch.AutoVoiced, nick)
}

func RemoveChannelAutoVoicedUser(e *irc.Event, ch *models.Channel, nick string) {
	autoVoiced := make([]string, 0)

	for _, n := range ch.AutoVoiced {
		if n != nick {
			autoVoiced = append(autoVoiced, n)
		}
	}

	ch.AutoVoiced = autoVoiced
}

func AddChannelAutoVoiceUser(e *irc.Event, ch *models.Channel, nick string) {
	if !slices.Contains(ch.AutoVoiced, nick) {
		ch.AutoVoiced = append(ch.AutoVoiced, nick)
	}
}

func UpdateChannelVoiceRequests(e *irc.Event, ch *models.Channel) error {
	logger := log.Logger()
	fs := firestore.Get()

	if err := fs.UpdateChannel(ch.Name, map[string]any{"voice_requests": ch.VoiceRequests, "updated_at": time.Now()}); err != nil {
		logger.Errorf(e, "error updating channel, %s", err)
		return err
	}

	return nil
}

func UpdateChannelAutoVoiced(e *irc.Event, ch *models.Channel) error {
	logger := log.Logger()
	fs := firestore.Get()

	if err := fs.UpdateChannel(ch.Name, map[string]any{"auto_voiced": ch.AutoVoiced, "updated_at": time.Now()}); err != nil {
		logger.Errorf(e, "error updating channel, %s", err)
		return err
	}

	return nil
}

func UpdateChannelDisabledCommands(e *irc.Event, ch *models.Channel) error {
	logger := log.Logger()
	fs := firestore.Get()

	if err := fs.UpdateChannel(ch.Name, map[string]any{"disabled_commands": ch.DisabledCommands, "updated_at": time.Now()}); err != nil {
		logger.Errorf(e, "error updating channel, %s", err)
		return err
	}

	return nil
}

func VoiceRequestExistsForNick(e *irc.Event, ch *models.Channel, nick string) bool {
	if ch.VoiceRequests == nil || len(ch.VoiceRequests) == 0 {
		return false
	}

	return slices.ContainsFunc(ch.VoiceRequests, func(request models.VoiceRequest) bool {
		return request.Nick == nick
	})
}

func VoiceRequestExistsForHost(e *irc.Event, ch *models.Channel, host string) bool {
	if ch.VoiceRequests == nil || len(ch.VoiceRequests) == 0 {
		return false
	}

	return slices.ContainsFunc(ch.VoiceRequests, func(request models.VoiceRequest) bool {
		return request.Host == host
	})
}

func AddChannelVoiceRequest(e *irc.Event, ch *models.Channel, mask *irc.Mask) {
	voiceRequests := make([]models.VoiceRequest, 0)
	for _, request := range ch.VoiceRequests {
		if request.Nick != mask.Nick {
			voiceRequests = append(voiceRequests, request)
		}
	}

	vr := models.VoiceRequest{
		Nick:        mask.Nick,
		Username:    mask.UserID,
		Host:        mask.Host,
		RequestedAt: time.Now(),
	}

	ch.VoiceRequests = append(ch.VoiceRequests, vr)
}

func ChannelVoiceRequestsForInput(e *irc.Event, ch *models.Channel, numbers []int) ([]models.VoiceRequest, error) {
	vrs := make([]models.VoiceRequest, 0)

	for _, number := range numbers {
		if number < 1 || number > len(ch.VoiceRequests) {
			return nil, fmt.Errorf("invalid voice request number: %d", number)
		}

		vr := ch.VoiceRequests[number-1]
		vrs = append(vrs, vr)
	}

	return vrs, nil
}

func RemoveChannelVoiceRequest(e *irc.Event, ch *models.Channel, nick, host string) {
	voiceRequests := make([]models.VoiceRequest, 0)

	for _, request := range ch.VoiceRequests {
		if (len(nick) == 0 || request.Nick != nick) && (len(host) == 0 || request.Host != host) {
			voiceRequests = append(voiceRequests, request)
		}
	}

	ch.VoiceRequests = voiceRequests
}

type quoteSearchResult struct {
	score int
	quote *models.Quote
}

func FindUserQuotesWithContent(channel, nick string, keywords []string) ([]*models.Quote, error) {
	fs := firestore.Get()
	matching, err := fs.FindUserQuotesWithContent(channel, nick, keywords)
	if err != nil {
		return nil, err
	}
	return rankQuoteSearchResults(matching, keywords)
}

func FindUserQuotes(channel, nick string) ([]*models.Quote, error) {
	fs := firestore.Get()
	return fs.FindUserQuotes(channel, nick)
}

func FindQuotes(channel string, keywords []string) ([]*models.Quote, error) {
	fs := firestore.Get()
	matching, err := fs.FindQuotes(channel, keywords)
	if err != nil {
		return nil, err
	}
	return rankQuoteSearchResults(matching, keywords)
}

func rankQuoteSearchResults(matching []*models.Quote, keywords []string) ([]*models.Quote, error) {
	topMatches := make([]*models.Quote, 0)

	sr := make([]quoteSearchResult, 0)
	for _, q := range matching {
		score := 0
		allMatch := true
		for _, k := range keywords {
			if strings.Contains(strings.ToLower(q.Quote), k) {
				score++
			} else {
				allMatch = false
			}
		}

		if allMatch {
			topMatches = append(topMatches, q)
		}

		if score > 0 {
			sr = append(sr, quoteSearchResult{score, q})
		}
	}

	if len(topMatches) > 0 {
		return topMatches, nil
	}

	sort.Slice(sr, func(i, j int) bool {
		return sr[i].score > sr[j].score
	})

	quotes := make([]*models.Quote, 0)
	for _, r := range sr {
		quotes = append(quotes, r.quote)
	}

	return quotes, nil
}
