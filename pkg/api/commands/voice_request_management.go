package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strconv"
	"strings"
)

const VoiceRequestManagementCommandName = "voice_request_management"
const maxVoiceRequestsToShow = 10

type VoiceRequestManagementCommand struct {
	*commandStub
}

func NewVoiceRequestManagementCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &VoiceRequestManagementCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *VoiceRequestManagementCommand) Name() string {
	return VoiceRequestManagementCommandName
}

func (c *VoiceRequestManagementCommand) Description() string {
	return "Manage voice requests for the specified channel. If no channel is specified, the current channel is used."
}

func (c *VoiceRequestManagementCommand) Triggers() []string {
	return []string{"voicerequests", "vr"}
}

func (c *VoiceRequestManagementCommand) Usages() []string {
	return []string{
		fmt.Sprintf("%%s (shows voice requests for channel, up to %d)", maxVoiceRequestsToShow),
		"%s <channel> (shows voice requests for specified channel)",
		"%s <channel> <y/n> <number> [<number>...] (approve [y] or deny [n] voice requests by number)",
	}
}

func (c *VoiceRequestManagementCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *VoiceRequestManagementCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *VoiceRequestManagementCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if e.IsPrivateMessage() {
		if len(tokens) > 1 && irc.IsChannel(tokens[1]) {
			channel = tokens[1]
			tokens = tokens[1:]
		} else {
			c.Replyf(e, "Please specify a channel to view voice requests for: %s", style.Italics(fmt.Sprintf("%s <channel>", tokens[0])))
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

		ch, err := repository.GetChannel(e, channel)
		if err != nil {
			logger.Errorf(e, "error retrieving channel, %s", err)
			return
		}

		if isApproveAction {
			vrs, err := repository.ChannelVoiceRequestsForInput(e, ch, numbers)
			if err != nil {
				logger.Errorf(e, "error retrieving voice requests, %s", err)
				return
			}
			for _, vr := range vrs {
				repository.RemoveChannelVoiceRequest(e, ch, vr.Nick, vr.Host)
				c.irc.Voice(channel, vr.Nick)
				c.Replyf(e, "Approved voice request for %s (%s)", style.Bold(vr.Nick), vr.Mask())
				c.SendMessage(e, vr.Nick, fmt.Sprintf("🎉 Your voice request in %s has been approved. Welcome! We'd love it if you'd take a moment to say hello and introduce yourself.", style.Bold(channel)))
				logger.Debugf(e, "approved voice request for %s (%s)", vr.Nick, vr.Mask())

				u, err := repository.GetUserByNick(e, channel, vr.Nick, false)
				if err != nil {
					logger.Errorf(e, "error getting user: %v", err)
					continue
				}

				if u != nil {
					u.IsAutoVoiced = true
					if err = repository.UpdateUserIsAutoVoiced(e, u); err != nil {
						logger.Errorf(e, "error updating user isAutoVoiced, %s", err)
					}
				}
			}
			if err = repository.UpdateChannelVoiceRequests(e, ch); err != nil {
				logger.Errorf(e, "error updating channel, %s", err)
			}

			return
		}

		if isDeclineAction {
			vrs, err := repository.ChannelVoiceRequestsForInput(e, ch, numbers)
			if err != nil {
				logger.Errorf(e, "error retrieving voice requests, %s", err)
				return
			}
			for _, vr := range vrs {
				repository.RemoveChannelVoiceRequest(e, ch, vr.Nick, vr.Host)
				c.Replyf(e, "Denied voice request for %s (%s)", style.Bold(vr.Nick), vr.Mask())
				logger.Debugf(e, "denied voice request for %s (%s)", vr.Nick, vr.Mask())
			}
			if err = repository.UpdateChannelVoiceRequests(e, ch); err != nil {
				logger.Errorf(e, "error updating channel, %s", err)
			}
			return
		}

		logger.Debugf(e, "showing voice requests for %s", channel)

		name := "requests"
		if len(ch.VoiceRequests) == 1 {
			name = "request"
		}

		messages := make([]string, 0)
		title := fmt.Sprintf("%s voice %s in %s", style.Bold(fmt.Sprintf("%d", len(ch.VoiceRequests))), name, style.Bold(channel))

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
	})
}
