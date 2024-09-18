package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
	"slices"
	"strings"
)

const (
	owner = "owner"
	admin = "admin"
)

type Function interface {
	IsAuthorized(e *core.Event, callback func(bool))
	MayExecute(e *core.Event) bool
	Execute(e *core.Event)
	Reply(e *core.Event, message string, args ...any)
}

func NewFunction(ctx context.Context, cfg *config.Config, irc core.IRC, name string) (Function, error) {
	switch name {
	case echoFunctionName:
		return NewEchoFunction(ctx, cfg, irc)
	case sayFunctionName:
		return NewSayFunction(ctx, cfg, irc)
	case helpFunctionName:
		return NewHelpFunction(ctx, cfg, irc)
	case joinFunctionName:
		return NewJoinFunction(ctx, cfg, irc)
	case leaveFunctionName:
		return NewLeaveFunction(ctx, cfg, irc)
	case uptimeFunctionName:
		return NewUptimeFunction(ctx, cfg, irc)
	case dateTimeFunctionName:
		return NewDateTimeFunction(ctx, cfg, irc)
	case kickFunctionName:
		return NewKickFunction(ctx, cfg, irc)
	case banFunctionName:
		return NewBanFunction(ctx, cfg, irc)
	case sleepFunctionName:
		return NewSleepFunction(ctx, cfg, irc)
	case wakeFunctionName:
		return NewWakeFunction(ctx, cfg, irc)
	case aboutFunctionName:
		return NewAboutFunction(ctx, cfg, irc)
	case searchFunctionName:
		return NewSearchFunction(ctx, cfg, irc)
	case "r/politics":
		return NewRedditFunction("politics", ctx, cfg, irc)
	case "r/news":
		return NewRedditFunction("news", ctx, cfg, irc)
	case "r/worldnews":
		return NewRedditFunction("worldnews", ctx, cfg, irc)
	case summaryFunctionName:
		return NewSummaryFunction(ctx, cfg, irc)
	case tempBanFunctionName:
		//return NewTempBanFunction(ctx, cfg, irc)
	}

	return nil, fmt.Errorf("unknown function: %s", name)
}

type Stub struct {
	ctx                 context.Context
	cfg                 *config.Config
	irc                 core.IRC
	Name                string
	Triggers            []string
	Description         string
	Usages              []string
	Role                string
	ChannelStatus       string
	DenyPrivateMessages bool
}

func newFunctionStub(ctx context.Context, cfg *config.Config, irc core.IRC, name string) (Stub, error) {
	entry, ok := cfg.Functions.EnabledFunctions[name]
	if !ok {
		return Stub{}, fmt.Errorf("no function named %s", name)
	}

	return Stub{
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

// isAwake checks if the bot is awake
func (f *Stub) isAwake() bool {
	return f.ctx.Value(context.IsAwakeKey).(bool)
}

// isUserAuthorizedByRole checks if the given sender is authorized based on authorization configuration settings
func (f *Stub) isUserAuthorizedByRole(user string, authorization string) bool {
	switch authorization {
	case owner:
		return user == f.cfg.Connection.Owner
	case admin:
		if user == f.cfg.Connection.Owner {
			return true
		}
		for _, a := range f.cfg.Connection.Admins {
			if user == a {
				return true
			}
		}
		return false
	}
	return true
}

// userChannelStatus retrieves the user's status in the channel (e.g., operator, half-operator, etc.)
func (f *Stub) userChannelStatus(user, channel string, callback func(string)) {
	f.irc.GetUserStatus(channel, user, func(status string) {
		callback(status)
	})
}

// isUserAuthorizedByChannelStatus checks if the given sender is authorized based on their channel status
func (f *Stub) isUserAuthorizedByChannelStatus(e *core.Event, required string, callback func(bool)) {
	user, _ := e.Sender()
	channel := e.ReplyTarget()

	f.userChannelStatus(user, channel, func(status string) {
		if !core.IsChannelStatusAtLeast(status, required) {
			callback(false)
			return
		}
		callback(true)
	})
}

// IsAuthorized checks authorization using both channel status-based and role-based authorization
func (f *Stub) IsAuthorized(e *core.Event, callback func(bool)) {
	if len(f.ChannelStatus) > 0 {
		f.isUserAuthorizedByChannelStatus(e, f.ChannelStatus, func(authorized bool) {
			if authorized {
				callback(true)
				return
			}

			if len(f.Role) > 0 {
				user, _ := e.Sender()
				if f.isUserAuthorizedByRole(user, f.Role) {
					callback(true)
					return
				}

				callback(false)
				return
			}

			callback(false)
		})
	} else if len(f.Role) > 0 {
		user, _ := e.Sender()
		if f.isUserAuthorizedByRole(user, f.Role) {
			callback(true)
			return
		}

		callback(false)
	} else {
		callback(true)
	}
}

// isTriggerValid checks if the given trigger is valid for the function
func (f *Stub) isTriggerValid(trigger string) bool {
	for _, t := range f.Triggers {
		if strings.TrimPrefix(trigger, f.cfg.Functions.Prefix) == t {
			return true
		}
	}

	return false
}

// isValid checks if the input meets minimum validation requirements for the function. If the function requires role-based authorization and not channel status-based authorization, then it is also checked. Otherwise, the union of both authorization methods will be checked during execution.
func (f *Stub) isValid(e *core.Event, minBodyTokens int) bool {
	user, _ := e.Sender()
	tokens := Tokens(e.Message())
	attempted := f.isTriggerValid(tokens[0])

	// if sleeping, ignore all triggers except wake
	if !f.isAwake() {
		isWakeTrigger := f.Name == wakeFunctionName && slices.Contains(f.functionConfig(wakeFunctionName).Triggers, strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix))
		if isWakeTrigger {
			if !f.isUserAuthorizedByRole(user, f.Role) {
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
			f.Reply(e, "The %s command is not allowed in private messages. See %s for more information.", text.Bold(strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix)), text.Italics(fmt.Sprintf("%s%s %s", f.cfg.Functions.Prefix, f.functionConfig(helpFunctionName).Triggers[0], strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix))))
		}
		return false
	}

	// if the function defines role-based authorization and not channel status-based authorization, check it
	if len(f.Role) > 0 && len(f.ChannelStatus) == 0 && !f.isUserAuthorizedByRole(user, f.Role) {
		if attempted {
			f.UnauthorizedReply(e)
		}
		return false
	}

	// if the function requires a minimum number of body Tokens, check that
	if minBodyTokens > 0 && len(tokens) < minBodyTokens+1 {
		if attempted {
			f.Reply(e, "Invalid number of arguments for %s. See %s for more information.", text.Bold(strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix)), text.Italics(fmt.Sprintf("%s%s %s", f.cfg.Functions.Prefix, f.functionConfig(helpFunctionName).Triggers[0], strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix))))
		}
		return false
	}

	// if function has no triggers, allow
	if len(f.Triggers) == 0 {
		return true
	}

	// if the function has function triggers but the input doesn't start with the function prefix, ignore
	if !strings.HasPrefix(tokens[0], f.cfg.Functions.Prefix) {
		return false
	}

	// if the trigger is valid, allow
	return attempted
}

// isBotAuthorizedByChannelStatus checks if the bot is authorized based on channel status
func (f *Stub) isBotAuthorizedByChannelStatus(channel string, required string, callback func(bool)) {
	f.userChannelStatus(f.cfg.Connection.Nick, channel, func(status string) {
		callback(core.IsChannelStatusAtLeast(status, required))
	})
}

// Reply sends a message to the Reply target
func (f *Stub) Reply(e *core.Event, message string, args ...any) {
	if !e.IsPrivateMessage() {
		message = e.From + ": " + strings.ToLower(message[0:1]) + message[1:]
	}

	if len(args) == 0 {
		f.irc.SendMessage(e.ReplyTarget(), message)
		return
	}

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf(message, args...))
}

func (f *Stub) UnauthorizedReply(e *core.Event) {
	tokens := Tokens(e.Message())
	f.Reply(e, "You are not authorized to use %s.", text.Bold(strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix)))
}

func (f *Stub) functionConfig(name string) config.FunctionConfig {
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

// splitMessageIfNecessary splits the input string into multiple messages if it exceeds the maximum length
func splitMessageIfNecessary(input string) []string {
	if len(input) < inputMaxLength {
		return []string{input}
	}

	messages := make([]string, 0)
	for len(input) > 0 {
		if len(input) > inputMaxLength {
			messages = append(messages, strings.TrimSpace(input[:inputMaxLength]))
			input = input[inputMaxLength:]
		} else {
			messages = append(messages, strings.TrimSpace(input))
			input = ""
		}
	}
	return messages
}
