package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const VoiceRequestsCommandName = "voice_requests"
const maxVoiceRequestsToShow = 10

type VoiceRequestsCommand struct {
	*commandStub
}

func NewVoiceRequestsCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &VoiceRequestsCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *VoiceRequestsCommand) Name() string {
	return VoiceRequestsCommandName
}

func (c *VoiceRequestsCommand) Description() string {
	return "Manage voice requests for the specified channel. If no channel is specified, the current channel is used."
}

func (c *VoiceRequestsCommand) Triggers() []string {
	return []string{"voicerequests", "vr"}
}

func (c *VoiceRequestsCommand) Usages() []string {
	return []string{
		"%s (shows voice requests for channel)",
		"%s <channel> (shows voice requests for specified channel)",
		"%s <y/n> <number> [<number>...] (approve [y] or deny [n] voice requests by number)",
	}
}

func (c *VoiceRequestsCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *VoiceRequestsCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *VoiceRequestsCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if e.IsPrivateMessage() {
		if len(tokens) > 1 && irc.IsChannel(tokens[1]) {
			channel = tokens[1]
			tokens = tokens[1:]
		} else {
			c.Replyf(e, "Please specify a channel to view voice requests for: %s <channel>", tokens[0])
			return
		}
	}

	action := ""
	numbers := make([]int, 0)
	if len(tokens) >= 2 {
		action = strings.ToLower(tokens[1])
		for _, token := range tokens[1:] {
			if n, err := strconv.Atoi(token); err == nil {
				numbers = append(numbers, n)
			}
		}
	}

	isManageAction := len(action) > 0 && len(numbers) > 0
	isApproveAction := isManageAction && (action == "y" || action == "yes")
	isDeclineAction := isManageAction && (action == "n" || action == "no")

	nick := e.From
	if e.IsPrivateMessage() {
		nick = e.ReplyTarget()
	}

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, nick)

	c.isBotAuthorizedByChannelStatus(channel, irc.ChannelStatusHalfOperator, func(authorized bool) {
		if !authorized {
			logger.Warningf(e, "bot lacks needed channel permissions in %s", channel)
			c.Replyf(e, "Missing required permissions to view voice requests in this channel. Did you forget /mode %s +h %s?", channel, c.cfg.IRC.Nick)
			return
		}

		fs := firestore.Get()
		ch, err := fs.Channel(channel)
		if err != nil {
			logger.Errorf(e, "error retrieving channel, %s", err)
			return
		}

		if ch == nil {
			logger.Errorf(e, "channel %s does not exist", channel)
			return
		}

		if ch.AutoVoiced == nil {
			ch.AutoVoiced = make([]string, 0)
		}

		if ch.VoiceRequests == nil {
			ch.VoiceRequests = make([]models.VoiceRequest, 0)
		}

		slices.SortFunc(ch.VoiceRequests, func(a, b models.VoiceRequest) int {
			return cmp.Compare(a.RequestedAt.Unix(), b.RequestedAt.Unix())
		})

		if !isManageAction {
			messages := make([]string, 0)
			title := fmt.Sprintf("%s voice requests in %s", style.Bold(fmt.Sprintf("%d", len(ch.VoiceRequests))), style.Bold(channel))

			vrs := ch.VoiceRequests
			if len(ch.VoiceRequests) > maxVoiceRequestsToShow {
				vrs = ch.VoiceRequests[:maxVoiceRequestsToShow]
				title += fmt.Sprintf(" (showing oldest %d)", maxVoiceRequestsToShow)
			}

			messages = append(messages, title)

			for i, vr := range vrs {
				messages = append(messages, fmt.Sprintf("%s: %s (%s), %s", style.Bold(style.Underline(fmt.Sprintf("%d", i+1))), style.Bold(vr.Nick), vr.Mask(), elapse.PastTimeDescription(vr.RequestedAt)))
			}

			c.SendMessages(e, e.ReplyTarget(), messages)
			return
		}

		if isApproveAction {
			vrs := make([]models.VoiceRequest, 0)
			for _, number := range numbers {
				if number < 1 || number > len(ch.VoiceRequests) {
					c.Replyf(e, "Invalid voice request number: %d", number)
					return
				}

				vr := ch.VoiceRequests[number-1]
				vrs = append(vrs, vr)
			}

			for _, vr := range vrs {
				if !slices.Contains(ch.AutoVoiced, vr.Nick) {
					ch.AutoVoiced = append(ch.AutoVoiced, vr.Nick)
				}

				voiceRequests := make([]models.VoiceRequest, 0)

				for _, request := range ch.VoiceRequests {
					if request.Nick != vr.Nick {
						voiceRequests = append(voiceRequests, request)
					}
				}

				ch.VoiceRequests = voiceRequests

				c.Replyf(e, "Approved voice request for %s (%s)", style.Bold(nick), vr.Mask())
			}

			if err = fs.UpdateChannel(ch.Name, map[string]any{"voice_requests": ch.VoiceRequests, "auto_voiced": ch.AutoVoiced}); err != nil {
				logger.Errorf(e, "error updating channel, %s", err)
				return
			}
		} else if isDeclineAction {
			vrs := make([]models.VoiceRequest, 0)
			for _, number := range numbers {
				if number < 1 || number > len(ch.VoiceRequests) {
					c.Replyf(e, "Invalid voice request number: %d", number)
					return
				}

				vr := ch.VoiceRequests[number-1]
				vrs = append(vrs, vr)
			}

			for _, vr := range vrs {
				voiceRequests := make([]models.VoiceRequest, 0)
				for _, request := range ch.VoiceRequests {
					if request.Nick != vr.Nick {
						voiceRequests = append(voiceRequests, request)
					}
				}

				ch.VoiceRequests = voiceRequests

				c.Replyf(e, "Denied voice request for %s (%s)", style.Bold(vr.Nick), vr.Mask())
			}

			if err = fs.UpdateChannel(ch.Name, map[string]any{"voice_requests": ch.VoiceRequests}); err != nil {
				logger.Errorf(e, "error updating channel, %s", err)
				return
			}
		} else {
			c.Replyf(e, "Invalid input, please specify %s to approve or %s to deny voice requests", style.Bold("y"), style.Bold("n"))
			return
		}

		voiceRequests := make([]models.VoiceRequest, 0)
		for _, request := range ch.VoiceRequests {
			if request.Nick != nick {
				voiceRequests = append(voiceRequests, request)
			}
		}

		ch.VoiceRequests = voiceRequests

		if err = fs.UpdateChannel(ch.Name, map[string]any{"voice_requests": ch.VoiceRequests, "auto_voiced": ch.AutoVoiced}); err != nil {
			logger.Errorf(e, "error updating channel, %s", err)
			return
		}
	})
}
