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
		f.irc.GetUserStatus(e.ReplyTarget(), sender, func(status string) {
			reply := make([]string, 0)
			reply = append(reply, fmt.Sprintf("%s: %s", text.Bold(text.Underline(f.Name)), f.Description))

			// create map of function name to slice of current user authorization and allowed user status
			all := make(map[string][]string)
			for fn, fs := range f.cfg.Functions.EnabledFunctions {
				for _, t := range f.cfg.Functions.EnabledFunctions[fn].Triggers {
					key := strings.TrimPrefix(t, f.cfg.Functions.Prefix)
					if len(all[key]) == 0 {
						all[key] = make([]string, 0)
					}
					all[key] = append(all[key], fs.Authorization)
					all[key] = append(all[key], fs.AllowedUserStatus)
				}
			}

			commands := make([]string, 0)
			for cmd, auths := range all {
				if f.isSenderAuthorized(sender, auths[0]) {
					commands = append(commands, cmd)
				} else if len(auths[1]) > 0 && core.IsUserStatusAtLeast(status, auths[1]) {
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
					reply = append(reply, fmt.Sprintf("Usage: %s", text.Italics(fmt.Sprintf(fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, u), t))))
				}
			}
			f.irc.SendMessages(e.ReplyTarget(), reply)
		})
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

	if len(fn.Triggers) == 0 {
		return nil
	}

	reply := make([]string, 0)
	reply = append(reply, fmt.Sprintf("%s: %s", text.Bold(text.Underline(tokens[1])), fn.Description))
	for _, u := range fn.Usages {
		for _, t := range fn.Triggers {
			reply = append(reply, fmt.Sprintf("Usage: %s", text.Italics(fmt.Sprintf(u, t))))
		}
	}
	if len(fn.Authorization) > 0 && len(fn.AllowedUserStatus) > 0 {
		reply = append(reply, fmt.Sprintf("Required: %s role, or %s or greater", fn.Authorization, core.OperatorStatusName(fn.AllowedUserStatus)))
	} else if len(fn.Authorization) > 0 {
		reply = append(reply, fmt.Sprintf("Required: %s role", fn.Authorization))
	} else if len(fn.AllowedUserStatus) > 0 {
		reply = append(reply, fmt.Sprintf("Required: %s or greater", core.OperatorStatusName(fn.AllowedUserStatus)))
	}

	f.irc.SendMessages(e.ReplyTarget(), reply)
	return nil
}
