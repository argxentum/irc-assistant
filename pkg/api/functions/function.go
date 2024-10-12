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

const (
	owner = "owner"
	admin = "admin"
)

type Function interface {
	IsAuthorized(e *irc.Event, channel string, callback func(bool))
	MayExecute(e *irc.Event) bool
	Execute(e *irc.Event)
	Replyf(e *irc.Event, message string, args ...any)
}

type FunctionStub struct {
	ctx                 context.Context
	cfg                 *config.Config
	irc                 irc.IRC
	Name                string
	Triggers            []string
	Description         string
	Usages              []string
	Role                string
	ChannelStatus       string
	DenyPrivateMessages bool
}

func newFunctionStub(ctx context.Context, cfg *config.Config, irc irc.IRC, name string) (FunctionStub, error) {
	entry, ok := cfg.Functions.EnabledFunctions[name]
	if !ok {
		return FunctionStub{}, fmt.Errorf("no function named %s", name)
	}

	return FunctionStub{
		ctx:                 ctx,
		cfg:                 cfg,
		irc:                 irc,
		Name:                name,
		Triggers:            entry.Triggers,
		Description:         entry.Description,
		Usages:              entry.Usages,
		Role:                entry.Role,
		ChannelStatus:       entry.ChannelStatus,
		DenyPrivateMessages: entry.DenyPrivateMessages,
	}, nil
}

const inputMaxLength = 512

// isUserAuthorizedByRole checks if the given sender is authorized based on authorization configuration settings
func (f *FunctionStub) isUserAuthorizedByRole(nick string, authorization string) bool {
	switch authorization {
	case owner:
		return nick == f.cfg.IRC.Owner
	case admin:
		if nick == f.cfg.IRC.Owner {
			return true
		}
		for _, a := range f.cfg.IRC.Admins {
			if nick == a {
				return true
			}
		}
		return false
	}
	return true
}

// userStatus retrieves the user's status in the channel (e.g., operator, half-operator, etc.)
func (f *FunctionStub) userStatus(channel, nick string, callback func(user *irc.User)) {
	f.irc.GetUser(channel, nick, callback)
}

func (f *FunctionStub) userStatuses(channel string, callback func([]irc.User)) {
	f.irc.GetUsers(channel, callback)
}

// isUserAuthorizedByChannelStatus checks if the given sender is authorized based on their channel status
func (f *FunctionStub) isUserAuthorizedByChannelStatus(e *irc.Event, channel, required string, callback func(bool)) {
	nick, _ := e.Sender()

	f.userStatus(channel, nick, func(user *irc.User) {
		if user != nil && !irc.IsStatusAtLeast(user.Status, required) {
			callback(false)
			return
		}
		callback(true)
	})
}

// IsAuthorized checks authorization using both channel status-based and role-based authorization
func (f *FunctionStub) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	if len(f.ChannelStatus) > 0 {
		f.isUserAuthorizedByChannelStatus(e, channel, f.ChannelStatus, func(authorized bool) {
			if authorized {
				callback(true)
				return
			}

			if len(f.Role) > 0 {
				nick, _ := e.Sender()
				if f.isUserAuthorizedByRole(nick, f.Role) {
					callback(true)
					return
				}

				callback(false)
				return
			}

			callback(false)
		})
	} else if len(f.Role) > 0 {
		nick, _ := e.Sender()
		if f.isUserAuthorizedByRole(nick, f.Role) {
			callback(true)
			return
		}

		callback(false)
	} else {
		callback(true)
	}
}

// isTriggerValid checks if the given trigger is valid for the function
func (f *FunctionStub) isTriggerValid(e *irc.Event, trigger string) bool {
	for _, t := range f.Triggers {
		if strings.TrimPrefix(trigger, f.cfg.Functions.Prefix) == t && (strings.HasPrefix(trigger, f.cfg.Functions.Prefix) || e.IsPrivateMessage()) {
			return true
		}
	}

	return false
}

func (f *FunctionStub) isValidForChannel(e *irc.Event, channel string, minBodyTokens int) bool {
	nick, _ := e.Sender()
	tokens := Tokens(e.Message())
	attempted := f.isTriggerValid(e, tokens[0])

	// if sleeping, ignore all triggers except wake
	if !f.ctx.Session().IsAwake {
		isWakeTrigger := f.Name == wakeFunctionName && slices.Contains(f.functionConfig(wakeFunctionName).Triggers, strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix))
		if isWakeTrigger {
			if !f.isUserAuthorizedByRole(nick, f.Role) {
				f.UnauthorizedReply(e)
				return false
			}
			return true
		}
		return false
	}

	// if the function is not allowed in private messages and the message is a private message, ignore
	if f.DenyPrivateMessages && e.IsPrivateMessage() {
		if attempted {
			f.Replyf(e, "The %s command is not allowed in private messages. See %s for more information.", style.Bold(strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix)), style.Italics(fmt.Sprintf("%s%s %s", f.cfg.Functions.Prefix, f.functionConfig(helpFunctionName).Triggers[0], strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix))))
		}
		return false
	}

	// if the function defines role-based authorization and not channel status-based authorization, check it
	if len(f.Role) > 0 && len(f.ChannelStatus) == 0 && !f.isUserAuthorizedByRole(nick, f.Role) {
		if attempted {
			f.UnauthorizedReply(e)
		}
		return false
	}

	// if the function requires a minimum number of body Tokens, check that
	if minBodyTokens > 0 && len(tokens) < minBodyTokens+1 {
		if attempted {
			f.Replyf(e, "Invalid number of arguments for %s. See %s for more information.", style.Bold(strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix)), style.Italics(fmt.Sprintf("%s%s %s", f.cfg.Functions.Prefix, f.functionConfig(helpFunctionName).Triggers[0], strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix))))
		}
		return false
	}

	// if function has no triggers, allow
	if len(f.Triggers) == 0 {
		return true
	}

	// if the function has function triggers but the input doesn't start with the function prefix and not in a private message, ignore
	if !e.IsPrivateMessage() && !strings.HasPrefix(tokens[0], f.cfg.Functions.Prefix) {
		return false
	}

	// if the trigger is valid, allow
	return attempted
}

// isValid checks if the input meets minimum validation requirements for the function. If the function requires role-based authorization and not channel status-based authorization, then it is also checked. Otherwise, the union of both authorization methods will be checked during execution.
func (f *FunctionStub) isValid(e *irc.Event, minBodyTokens int) bool {
	return f.isValidForChannel(e, e.ReplyTarget(), minBodyTokens)
}

// isBotAuthorizedByChannelStatus checks if the bot is authorized based on channel status
func (f *FunctionStub) isBotAuthorizedByChannelStatus(channel string, required string, callback func(bool)) {
	f.userStatus(channel, f.cfg.IRC.Nick, func(user *irc.User) {
		if user != nil {
			callback(irc.IsStatusAtLeast(user.Status, required))
		} else {
			callback(false)
		}
	})
}

func (f *FunctionStub) SendMessage(e *irc.Event, target, message string) {
	log.Logger().Infof(e, "Sending message to %s: %s", target, message)
	f.irc.SendMessage(target, message)
}

func (f *FunctionStub) SendMessages(e *irc.Event, target string, messages []string) {
	log.Logger().Infof(e, "Sending messages to %s: [%s]", target, strings.Join(messages, ", "))
	f.irc.SendMessages(target, messages)
}

// Replyf sends a message to the reply target
func (f *FunctionStub) Replyf(e *irc.Event, message string, args ...any) {
	log.Logger().Infof(e, "Replying: %s", fmt.Sprintf(message, args...))

	if !e.IsPrivateMessage() {
		message = fmt.Sprintf("%s: %s", e.From, text.Uncapitalize(message, false))
	}

	if len(args) == 0 {
		f.irc.SendMessage(e.ReplyTarget(), message)
		return
	}

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf(message, args...))
}

func (f *FunctionStub) UnauthorizedReply(e *irc.Event) {
	tokens := Tokens(e.Message())
	f.Replyf(e, "You are not authorized to use %s.", style.Bold(strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix)))
}

func (f *FunctionStub) functionConfig(name string) config.FunctionConfig {
	return f.cfg.Functions.EnabledFunctions[name]
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
