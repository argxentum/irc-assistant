package functions

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

type Function interface {
	Name() string
	Description() string
	Triggers() []string
	Usages() []string
	AllowedInPrivateMessages() bool
	Authorizer() FunctionAuthorizer
	CanExecute(e *irc.Event) bool
	Execute(e *irc.Event)
	Replyf(e *irc.Event, message string, args ...any)
}

const inputMaxLength = 512

type functionStub struct {
	ctx        context.Context
	cfg        *config.Config
	irc        irc.IRC
	authorizer FunctionAuthorizer
}

func newFunctionStub(ctx context.Context, cfg *config.Config, ircs irc.IRC, requiredRole Role, requiredChannelStatus irc.ChannelStatus) *functionStub {
	return &functionStub{
		ctx:        ctx,
		cfg:        cfg,
		irc:        ircs,
		authorizer: newFunctionAuthorizer(ctx, cfg, ircs, requiredRole, requiredChannelStatus),
	}
}

func defaultFunctionStub(ctx context.Context, cfg *config.Config, ircs irc.IRC) *functionStub {
	return newFunctionStub(ctx, cfg, ircs, RoleUnprivileged, irc.ChannelStatusNormal)
}

func (fs *functionStub) Authorizer() FunctionAuthorizer {
	return fs.authorizer
}

func (fs *functionStub) isTriggerValid(f Function, e *irc.Event, trigger string) bool {
	for _, t := range f.Triggers() {
		if strings.TrimPrefix(trigger, fs.cfg.Functions.Prefix) == t && (strings.HasPrefix(trigger, fs.cfg.Functions.Prefix) || e.IsPrivateMessage()) {
			return true
		}
	}

	return false
}

func (fs *functionStub) isFunctionEventValid(f Function, e *irc.Event, minBodyTokens int) bool {
	nick, _ := e.Sender()
	tokens := Tokens(e.Message())
	attempted := fs.isTriggerValid(f, e, tokens[0])

	// if sleeping, ignore all triggers except wake
	if !fs.ctx.Session().IsAwake {
		isWakeTrigger := f.Name() == wakeFunctionName && slices.Contains(registry.Function(wakeFunctionName).Triggers(), strings.TrimPrefix(tokens[0], fs.cfg.Functions.Prefix))
		if isWakeTrigger {
			if !fs.authorizer.IsUserAuthorizedByRole(nick, fs.authorizer.RequiredRole()) {
				fs.UnauthorizedReply(e)
				return false
			}
			return true
		}
		return false
	}

	// if the functionStub is not allowed in private messages and the message is a private message, ignore
	if !f.AllowedInPrivateMessages() && e.IsPrivateMessage() {
		if attempted {
			fs.Replyf(e, "The %s command is not allowed in private messages. See %s for more information.", style.Bold(strings.TrimPrefix(tokens[0], fs.cfg.Functions.Prefix)), style.Italics(fmt.Sprintf("%s%s %s", fs.cfg.Functions.Prefix, registry.Function(helpFunctionName).Triggers()[0], strings.TrimPrefix(tokens[0], fs.cfg.Functions.Prefix))))
		}
		return false
	}

	// if the functionStub defines role-based authorization and not channel status-based authorization, check it
	if len(fs.authorizer.RequiredRole()) > 0 && len(fs.authorizer.RequiredChannelStatus()) == 0 && !fs.authorizer.IsUserAuthorizedByRole(nick, fs.authorizer.RequiredRole()) {
		if attempted {
			fs.UnauthorizedReply(e)
		}
		return false
	}

	// if the functionStub requires a minimum number of body Tokens, check that
	if minBodyTokens > 0 && len(tokens) < minBodyTokens+1 {
		if attempted {
			fs.Replyf(e, "Invalid number of arguments for %s. See %s for more information.", style.Bold(strings.TrimPrefix(tokens[0], fs.cfg.Functions.Prefix)), style.Italics(fmt.Sprintf("%s%s %s", fs.cfg.Functions.Prefix, registry.Function(helpFunctionName).Triggers()[0], strings.TrimPrefix(tokens[0], fs.cfg.Functions.Prefix))))
		}
		return false
	}

	// if functionStub has no triggers, allow
	if len(f.Triggers()) == 0 {
		return true
	}

	// if the functionStub has functionStub triggers but the input doesn't start with the functionStub prefix and not in a private message, ignore
	if !e.IsPrivateMessage() && !strings.HasPrefix(tokens[0], fs.cfg.Functions.Prefix) {
		return false
	}

	// if the trigger is valid, allow
	return attempted
}

// isBotAuthorizedByChannelStatus checks if the bot is authorized based on channel status
func (fs *functionStub) isBotAuthorizedByChannelStatus(channel string, status irc.ChannelStatus, callback func(bool)) {
	fs.authorizer.UserStatus(channel, fs.cfg.IRC.Nick, func(user *irc.User) {
		if user != nil {
			callback(irc.IsStatusAtLeast(user.Status, status))
		} else {
			callback(false)
		}
	})
}

func (fs *functionStub) Join(channel string) {
	log.Logger().Infof(nil, "joining %s", channel)
	fs.irc.Join(channel)
}

func (fs *functionStub) Leave(channel string) {
	log.Logger().Infof(nil, "leaving %s", channel)
	fs.irc.Part(channel)
}

func (fs *functionStub) SendMessage(e *irc.Event, target, message string) {
	log.Logger().Infof(e, "Sending message to %s: %s", target, message)
	fs.irc.SendMessage(target, message)
}

func (fs *functionStub) SendMessages(e *irc.Event, target string, messages []string) {
	log.Logger().Infof(e, "Sending messages to %s: [%s]", target, strings.Join(messages, ", "))
	fs.irc.SendMessages(target, messages)
}

func (fs *functionStub) Replyf(e *irc.Event, message string, args ...any) {
	log.Logger().Infof(e, "Replying: %s", fmt.Sprintf(message, args...))

	if !e.IsPrivateMessage() {
		message = fmt.Sprintf("%s: %s", e.From, text.Uncapitalize(message, false))
	}

	if len(args) == 0 {
		fs.irc.SendMessage(e.ReplyTarget(), message)
		return
	}

	fs.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf(message, args...))
}

func (fs *functionStub) UnauthorizedReply(e *irc.Event) {
	tokens := Tokens(e.Message())
	fs.Replyf(e, "You are not authorized to use %s.", style.Bold(strings.TrimPrefix(tokens[0], fs.cfg.Functions.Prefix)))
}

// sanitize cleans the input string
func sanitize(input string) string {
	sanitized := strings.TrimSpace(input)
	if len(sanitized) > inputMaxLength {
		return sanitized[:inputMaxLength]
	}
	return sanitized
}

// Tokens splits the input string into sanitized Tokens
func Tokens(input string) []string {
	return strings.Split(sanitize(input), " ")
}

func coalesce(strings ...string) string {
	for _, s := range strings {
		if len(s) > 0 {
			return s
		}
	}
	return ""
}
