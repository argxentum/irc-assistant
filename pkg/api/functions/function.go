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
	Matches(e *core.Event) bool
	Execute(e *core.Event) error
}

func NewFunction(ctx context.Context, cfg *config.Config, irc core.IRC, name string) (Function, error) {
	switch name {
	case echoFunctionName:
		return NewEchoFunction(ctx, cfg, irc)
	case helpFunctionName:
		return NewHelpFunction(ctx, cfg, irc)
	case joinFunctionName:
		return NewJoinFunction(ctx, cfg, irc)
	case uptimeFunctionName:
		return NewUptimeFunction(ctx, cfg, irc)
	}

	return nil, fmt.Errorf("unknown function: %s", name)
}

type stub struct {
	ctx           context.Context
	cfg           *config.Config
	irc           core.IRC
	Name          string
	Prefix        string
	Description   string
	Usage         []string
	Authorization string
}

func newFunctionStub(ctx context.Context, cfg *config.Config, irc core.IRC, name string) (stub, error) {
	entry, ok := cfg.Functions.Enabled[name]
	if !ok {
		return stub{}, fmt.Errorf("no function named %s", name)
	}

	return stub{
		ctx:           ctx,
		cfg:           cfg,
		irc:           irc,
		Name:          name,
		Prefix:        entry.Prefix,
		Description:   entry.Description,
		Usage:         entry.Usage,
		Authorization: entry.Authorization,
	}, nil
}

func sanitized(input string, n int) string {
	sanitized := strings.Trim(input, " \t")
	if n > 0 && len(sanitized) > n {
		return sanitized[:n]
	}
	return sanitized
}

func sanitizedTokens(input string, n int) []string {
	return strings.Split(sanitized(input, n), " ")
}

func (f *stub) isAuthorized(e *core.Event) bool {
	sender, _ := e.Sender()

	switch f.Authorization {
	case owner:
		return sender == f.cfg.Connection.Owner
	case admin:
		if sender == f.cfg.Connection.Owner {
			return true
		}
		for _, admin := range f.cfg.Connection.Admins {
			if sender == admin {
				return true
			}
		}
		return false
	}

	return true
}
