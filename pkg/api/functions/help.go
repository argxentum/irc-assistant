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
	Stub
}

func NewHelpFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, helpFunctionName)
	if err != nil {
		return nil, err
	}

	return &helpFunction{
		Stub: stub,
	}, nil
}

func (f *helpFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 0)
}

func (f *helpFunction) Execute(e *core.Event) {
	// if no command is specified, list all available commands for the user (based on their status)
	tokens := Tokens(e.Message())
	if len(tokens) == 1 {
		reply := make([]string, 0)
		reply = append(reply, fmt.Sprintf("%s: %s", text.Bold(text.Underline(f.Name)), f.Description))

		// create map of function name to slice of current user authorization and allowed user status
		commands := make([]string, 0)
		for fn := range f.cfg.Functions.EnabledFunctions {
			for _, t := range f.cfg.Functions.EnabledFunctions[fn].Triggers {
				key := strings.TrimPrefix(t, f.cfg.Functions.Prefix)
				commands = append(commands, key)
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
		reply = append(reply, splitMessageIfNecessary(fmt.Sprintf("Commands: %s", fns))...)

		for _, u := range f.Usages {
			for _, t := range f.Triggers {
				reply = append(reply, fmt.Sprintf("   %s", text.Italics(fmt.Sprintf(fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, u), t))))
			}
		}

		f.irc.SendMessages(e.ReplyTarget(), reply)
		return
	}

	found := false
	var fn config.FunctionConfig
	for _, s := range f.cfg.Functions.EnabledFunctions {
		for _, t := range s.Triggers {
			if tokens[1] == t {
				found = true
				fn = s
			}
		}
	}

	if !found {
		f.Reply(e, "Command %s not found. See %s for a list of available commands.", text.Bold(tokens[1]), text.Italics(fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, f.cfg.Functions.EnabledFunctions[helpFunctionName].Triggers[0])))
		return
	}

	if len(fn.Triggers) == 0 {
		return
	}

	reply := make([]string, 0)
	reply = append(reply, fmt.Sprintf("%s: %s", text.Bold(text.Underline(tokens[1])), fn.Description))
	for _, u := range fn.Usages {
		for _, t := range fn.Triggers {
			reply = append(reply, fmt.Sprintf("   %s", text.Italics(fmt.Sprintf(f.cfg.Functions.Prefix+u, t))))
		}
	}

	if len(fn.Role) > 0 && len(fn.ChannelStatus) > 0 {
		reply = append(reply, fmt.Sprintf("Requires %s role or %s status or greater.", fn.Role, core.ChannelStatusName(fn.ChannelStatus)))
	} else if len(fn.Role) > 0 {
		reply = append(reply, fmt.Sprintf("Requires %s role.", fn.Role))
	} else if len(fn.ChannelStatus) > 0 {
		reply = append(reply, fmt.Sprintf("Requires %s status or greater.", core.ChannelStatusName(fn.ChannelStatus)))
	}

	if fn.DenyPrivateMessages {
		reply = append(reply, "Must be used in a channel.")
	}

	f.irc.SendMessages(e.ReplyTarget(), reply)
}
