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

func (f *helpFunction) ShouldExecute(e *core.Event) bool {
	ok, _ := f.verifyInput(e, 0)
	return ok
}

func (f *helpFunction) Execute(e *core.Event) error {
	sender, _ := e.Sender()
	tokens := parseTokens(e.Message())

	if len(tokens) == 1 {
		reply := make([]string, 0)
		reply = append(reply, fmt.Sprintf("%s: %s", text.Bold(text.Underline(f.Name)), f.Description))

		all := make(map[string]string)
		for fn, fs := range f.cfg.Functions.EnabledFunctions {
			for _, t := range f.cfg.Functions.EnabledFunctions[fn].Triggers {
				all[strings.TrimPrefix(t, "!")] = fs.Authorization
			}
		}

		commands := make([]string, 0)
		for cmd, auth := range all {
			if f.isSenderAuthorized(sender, auth) {
				commands = append(commands, cmd)
			}
		}
		slices.Sort(commands)

		fns := ""
		for _, cmd := range commands {
			if len(fns) > 0 {
				fns += ", "
			}
			fns += cmd
		}
		reply = append(reply, splitMessageIfNecessary(fmt.Sprintf("Your available commands: %s", fns))...)

		for _, u := range f.Usages {
			for _, t := range f.Triggers {
				reply = append(reply, fmt.Sprintf("Usage: %s", text.Italics(fmt.Sprintf(u, t))))
			}
		}
		f.irc.SendMessages(e.ReplyTarget(), reply)
		return nil
	}

	var fn config.FunctionConfig
	for _, s := range f.cfg.Functions.EnabledFunctions {
		for _, t := range s.Triggers {
			if tokens[1] == t {
				fn = s
			}
		}
	}

	reply := make([]string, 0)
	reply = append(reply, fmt.Sprintf("%s: %s", text.Bold(text.Underline(tokens[1])), fn.Description))
	for _, u := range fn.Usages {
		for _, t := range fn.Triggers {
			reply = append(reply, fmt.Sprintf("Usage: %s", text.Italics(fmt.Sprintf(u, t))))
		}
	}
	if len(fn.Authorization) > 0 {
		reply = append(reply, fmt.Sprintf("Required role: %s", fn.Authorization))
	}

	f.irc.SendMessages(e.ReplyTarget(), reply)
	return nil
}
