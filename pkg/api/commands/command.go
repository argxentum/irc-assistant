package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"slices"
	"strings"
)

type Role string

const (
	RoleOwner        Role = "owner"
	RoleAdmin        Role = "admin"
	RoleUnprivileged Role = ""
)

type Command interface {
	Name() string
	Description() string
	Triggers() []string
	Usages() []string
	AllowedInPrivateMessages() bool
	Authorizer() CommandAuthorizer
	CanExecute(e *irc.Event) bool
	Execute(e *irc.Event)
	Replyf(e *irc.Event, message string, args ...any)
}

type commandStub struct {
	ctx        context.Context
	cfg        *config.Config
	irc        irc.IRC
	authorizer CommandAuthorizer
}

func newCommandStub(ctx context.Context, cfg *config.Config, ircs irc.IRC, requiredRole Role, requiredChannelStatus irc.ChannelStatus) *commandStub {
	return &commandStub{
		ctx:        ctx,
		cfg:        cfg,
		irc:        ircs,
		authorizer: newCommandAuthorizer(ctx, cfg, ircs, requiredRole, requiredChannelStatus),
	}
}

func defaultCommandStub(ctx context.Context, cfg *config.Config, ircs irc.IRC) *commandStub {
	return newCommandStub(ctx, cfg, ircs, RoleUnprivileged, irc.ChannelStatusNone)
}

func (cs *commandStub) Authorizer() CommandAuthorizer {
	return cs.authorizer
}

func (cs *commandStub) isTriggerValid(c Command, e *irc.Event, trigger string) bool {
	for _, t := range c.Triggers() {
		if strings.TrimPrefix(trigger, cs.cfg.Commands.Prefix) == t && (strings.HasPrefix(trigger, cs.cfg.Commands.Prefix) || e.IsPrivateMessage()) {
			return true
		}
	}

	return false
}

func (cs *commandStub) isCommandEventValid(c Command, e *irc.Event, minBodyTokens int) bool {
	nick, _ := e.Sender()
	tokens := Tokens(e.Message())
	attempted := cs.isTriggerValid(c, e, tokens[0])

	// if sleeping, ignore all triggers except wake
	if !cs.ctx.Session().IsAwake {
		isWakeTrigger := c.Name() == WakeCommandName && slices.Contains(registry.Command(WakeCommandName).Triggers(), strings.TrimPrefix(tokens[0], cs.cfg.Commands.Prefix))
		if isWakeTrigger {
			if !cs.authorizer.IsUserAuthorizedByRole(nick, cs.authorizer.RequiredRole()) {
				cs.UnauthorizedReply(e)
				return false
			}
			return true
		}
		return false
	}

	// if the commandStub is not allowed in private messages and the message is a private message, ignore
	if !c.AllowedInPrivateMessages() && e.IsPrivateMessage() {
		if attempted {
			cs.Replyf(e, "The %s command is not allowed in private messages. See %s for more information.", style.Bold(strings.TrimPrefix(tokens[0], cs.cfg.Commands.Prefix)), style.Italics(fmt.Sprintf("%s%s %s", cs.cfg.Commands.Prefix, registry.Command(HelpCommandName).Triggers()[0], strings.TrimPrefix(tokens[0], cs.cfg.Commands.Prefix))))
		}
		return false
	}

	// if the commandStub defines role-based authorization and not channel status-based authorization, check it
	if len(cs.authorizer.RequiredRole()) > 0 && len(cs.authorizer.RequiredChannelStatus()) == 0 && !cs.authorizer.IsUserAuthorizedByRole(nick, cs.authorizer.RequiredRole()) {
		if attempted {
			cs.UnauthorizedReply(e)
		}
		return false
	}

	// if the commandStub requires a minimum number of body Tokens, check that
	if minBodyTokens > 0 && len(tokens) < minBodyTokens+1 {
		if attempted {
			cs.Replyf(e, "Invalid number of arguments for %s. See %s for more information.", style.Bold(strings.TrimPrefix(tokens[0], cs.cfg.Commands.Prefix)), style.Italics(fmt.Sprintf("%s%s %s", cs.cfg.Commands.Prefix, registry.Command(HelpCommandName).Triggers()[0], strings.TrimPrefix(tokens[0], cs.cfg.Commands.Prefix))))
		}
		return false
	}

	// if commandStub has no triggers, allow
	if len(c.Triggers()) == 0 {
		return true
	}

	// if the commandStub has commandStub triggers but the input doesn't start with the commandStub prefix and not in a private message, ignore
	if !e.IsPrivateMessage() && !strings.HasPrefix(tokens[0], cs.cfg.Commands.Prefix) {
		return false
	}

	// if the trigger is valid, allow
	return attempted
}

// isBotAuthorizedByChannelStatus checks if the bot is authorized based on channel status
func (cs *commandStub) isBotAuthorizedByChannelStatus(channel string, status irc.ChannelStatus, callback func(bool)) {
	cs.authorizer.ListUsers(channel, func(users []*irc.User) {
		for _, user := range users {
			if user.Mask.Nick == cs.cfg.IRC.Nick {
				callback(irc.IsStatusAtLeast(user.Status, status))
				return
			}
		}
		callback(false)
	})
}

func (cs *commandStub) Join(channel string) {
	log.Logger().Infof(nil, "joining %s", channel)
	cs.irc.Join(channel)
}

func (cs *commandStub) Leave(channel string) {
	log.Logger().Infof(nil, "leaving %s", channel)
	cs.irc.Part(channel)
}

func (cs *commandStub) SendMessage(e *irc.Event, target, message string) {
	log.Logger().Infof(e, "Sending message to %s: %s", target, message)
	cs.irc.SendMessage(target, message)
}

func (cs *commandStub) SendMessages(e *irc.Event, target string, messages []string) {
	log.Logger().Infof(e, "Sending messages to %s: [%s]", target, strings.Join(messages, ", "))
	cs.irc.SendMessages(target, messages)
}

func (cs *commandStub) Replyf(e *irc.Event, message string, args ...any) {
	log.Logger().Infof(e, "Replying: %s", fmt.Sprintf(message, args...))

	if !e.IsPrivateMessage() {
		message = fmt.Sprintf("%s: %s", e.From, text.Uncapitalize(message, false))
	}

	if len(args) == 0 {
		cs.irc.SendMessage(e.ReplyTarget(), message)
		return
	}

	cs.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf(message, args...))
}

func (cs *commandStub) UnauthorizedReply(e *irc.Event) {
	tokens := Tokens(e.Message())
	cs.Replyf(e, "You are not authorized to use %s.", style.Bold(strings.TrimPrefix(tokens[0], cs.cfg.Commands.Prefix)))
}

func (cs *commandStub) ExecuteSynthesizedEvent(orig *irc.Event, command, payload string) {
	cmd := registry.Command(command)
	args := orig.Arguments
	args[1] = cs.cfg.Commands.Prefix + cmd.Triggers()[0] + " " + payload

	modified := &irc.Event{
		ID:        orig.ID,
		Raw:       fmt.Sprintf("%s %s", command, args[1]),
		Code:      orig.Code,
		From:      orig.From,
		Source:    orig.Source,
		Arguments: args,
	}

	cmd.Execute(modified)
}

// Tokens splits the input string into sanitized Tokens
func Tokens(input string) []string {
	return strings.Split(text.SanitizeToMaxLength(input, 512), " ")
}

func coalesce(strings ...string) string {
	for _, s := range strings {
		if len(s) > 0 {
			return s
		}
	}
	return ""
}
