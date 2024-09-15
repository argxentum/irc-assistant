package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
	"strings"
)

const (
	owner = "owner"
	admin = "admin"
)

type Function interface {
	ShouldExecute(e *core.Event) bool
	Execute(e *core.Event) error
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
	}

	return nil, fmt.Errorf("unknown function: %s", name)
}

type stub struct {
	ctx           context.Context
	cfg           *config.Config
	irc           core.IRC
	Name          string
	Triggers      []string
	Description   string
	Usages        []string
	Authorization string
}

func newFunctionStub(ctx context.Context, cfg *config.Config, irc core.IRC, name string) (stub, error) {
	entry, ok := cfg.Functions.EnabledFunctions[name]
	if !ok {
		return stub{}, fmt.Errorf("no function named %s", name)
	}

	return stub{
		ctx:           ctx,
		cfg:           cfg,
		irc:           irc,
		Name:          name,
		Triggers:      entry.Triggers,
		Description:   entry.Description,
		Usages:        entry.Usages,
		Authorization: entry.Authorization,
	}, nil
}

const inputMaxLength = 512

func sanitize(input string) string {
	sanitized := strings.TrimSpace(input)
	if len(sanitized) > inputMaxLength {
		return sanitized[:inputMaxLength]
	}
	return sanitized
}

func parseTokens(input string) []string {
	return strings.Split(sanitize(input), " ")
}

// isSenderAuthorized checks if the given sender has the given authorization level
func (f *stub) isSenderAuthorized(sender string, authorization string) bool {
	switch authorization {
	case owner:
		return sender == f.cfg.Connection.Owner
	case admin:
		if sender == f.cfg.Connection.Owner {
			return true
		}
		for _, a := range f.cfg.Connection.Admins {
			if sender == a {
				return true
			}
		}
		return false
	}
	return true
}

// verifyInput parses the event tokens and checks that the user has necessary authorization, the trigger corresponds to an enabled function, and the message has at least minBodyTokens (0 if not required)
func (f *stub) verifyInput(e *core.Event, minBodyTokens int) (bool, []string) {
	sender, _ := e.Sender()
	if !f.isSenderAuthorized(sender, f.Authorization) {
		return false, []string{}
	}

	tokens := parseTokens(e.Message())
	if minBodyTokens > 0 && len(tokens) < minBodyTokens+1 {
		return false, tokens
	}

	for _, t := range f.Triggers {
		if strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix) == t {
			return true, tokens
		}
	}
	return false, tokens
}

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
