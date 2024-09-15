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

const helpFunctionName = "help"

type helpFunction struct {
	stub
}

func NewHelpFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, helpFunctionName)
	if err != nil {
		return nil, err
	}

	return &helpFunction{
		stub: stub,
	}, nil
}

func (f *helpFunction) Matches(e *core.Event) bool {
	if !f.isAuthorized(e) {
		return false
	}

	tokens := sanitizedTokens(e.Message(), 200)
	if len(tokens) == 0 {
		return false
	}

	for _, p := range f.Prefixes {
		if tokens[0] == p {
			return true
		}
	}
	return false
}

func (f *helpFunction) Execute(e *core.Event) error {
	tokens := sanitizedTokens(e.Message(), 200)
	if len(tokens) == 1 {
		reply := make([]string, 0)
		reply = append(reply, fmt.Sprintf("%s: %s", text.Bold(text.Underline(f.Name)), f.Description))

		commands := make([]string, 0)
		for k, _ := range f.cfg.Functions.Enabled {
			commands = append(commands, k)
		}
		slices.Sort(commands)

		fns := ""
		for _, cmd := range commands {
			if len(fns) > 0 {
				fns += ", "
			}
			fns += cmd
		}
		reply = append(reply, fmt.Sprintf("Available commands: %s", fns))

		for _, u := range f.Usage {
			for _, p := range f.Prefixes {
				reply = append(reply, fmt.Sprintf("Usage: %s", text.Italics(fmt.Sprintf(u, p))))
			}
		}
		f.irc.SendMessages(e.ReplyTarget(), reply)
		return nil
	}

	fn, _ := f.cfg.Functions.Enabled[tokens[1]]
	reply := make([]string, 0)
	reply = append(reply, fmt.Sprintf("%s: %s", text.Bold(text.Underline(tokens[1])), fn.Description))
	for _, u := range fn.Usage {
		for _, p := range strings.Split(fn.Prefix, ", ") {
			reply = append(reply, fmt.Sprintf("Usage: %s", text.Italics(fmt.Sprintf(u, p))))
		}
	}
	if len(fn.Authorization) > 0 {
		reply = append(reply, fmt.Sprintf("Required role: %s", fn.Authorization))
	}

	f.irc.SendMessages(e.ReplyTarget(), reply)
	return nil
}
