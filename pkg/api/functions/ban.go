package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"strings"
)

const banFunctionName = "ban"

type banFunction struct {
	stub
}

func NewBanFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, banFunctionName)
	if err != nil {
		return nil, err
	}

	return &banFunction{
		stub: stub,
	}, nil
}

func (f *banFunction) ShouldExecute(e *core.Event) bool {
	if e.IsPrivateMessage() {
		return false
	}

	tokens := parseTokens(e.Message())
	if len(tokens) < 2 {
		return false
	}

	for _, t := range f.Triggers {
		if strings.TrimPrefix(tokens[0], f.cfg.Functions.Prefix) == t {
			return true
		}
	}
	return false
}

func (f *banFunction) Execute(e *core.Event) error {
	sender, _ := e.Sender()
	f.irc.GetUserStatus(e.ReplyTarget(), sender, func(status string) {
		if !core.IsUserStatusAtLeast(status, f.AllowedUserStatus) && !f.isSenderAuthorized(sender, f.Authorization) {
			return
		}

		tokens := parseTokens(e.Message())
		channel := e.ReplyTarget()
		user := tokens[1]
		reason := ""
		if len(tokens) > 2 {
			reason = strings.Join(tokens[2:], " ")
		}
		f.irc.Ban(channel, user, reason)
	})

	return nil
}
