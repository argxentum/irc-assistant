package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"slices"
	"strings"
)

const helpFunctionName = "help"

type helpFunction struct {
	FunctionStub
}

func NewHelpFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, helpFunctionName)
	if err != nil {
		return nil, err
	}

	return &helpFunction{
		FunctionStub: stub,
	}, nil
}

func (f *helpFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 0)
}

func (f *helpFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] help - %s", e.From, e.ReplyTarget(), e.Message())

	// if no command is specified, list all available commands
	if len(tokens) == 1 {
		reply := make([]string, 0)
		reply = append(reply, fmt.Sprintf("%s: %s", style.Bold(style.Underline(f.Name)), f.Description))

		// create map of function name to slice of current user authorization and allowed user status
		commands := make([]string, 0)
		for fn := range f.cfg.Functions.EnabledFunctions {
			fnt := ""
			for i, t := range f.functionConfig(fn).Triggers {
				if len(fnt) > 0 {
					fnt += "/"
				}

				if i == len(f.functionConfig(fn).Triggers)-1 && (len(f.cfg.Functions.EnabledFunctions[fn].Role) > 0 || len(f.cfg.Functions.EnabledFunctions[fn].ChannelStatus) > 0) {
					fnt += fmt.Sprintf("%s*", strings.TrimPrefix(t, f.cfg.Functions.Prefix))
				} else {
					fnt += strings.TrimPrefix(t, f.cfg.Functions.Prefix)
				}
			}
			commands = append(commands, fnt)
		}
		slices.Sort(commands)

		fns := ""
		for _, cmd := range commands {
			if len(fns) > 0 {
				fns += ", "
			}
			fns += cmd
		}
		reply = append(reply, fmt.Sprintf("%s: %s (* requires authorization)", style.Underline("Commands"), fns))
		usages := ""
		for _, u := range f.Usages {
			if len(usages) > 0 {
				usages += ", "
			}
			usages += fmt.Sprintf(u, fmt.Sprintf("%shelp", f.cfg.Functions.Prefix))
		}
		reply = append(reply, fmt.Sprintf("%s: %s", style.Underline("Usage"), style.Italics(usages)))

		f.SendMessages(e, e.ReplyTarget(), reply)
		return
	}

	trigger := strings.TrimPrefix(tokens[1], f.cfg.Functions.Prefix)

	found := false
	var fn config.FunctionConfig
	for _, s := range f.cfg.Functions.EnabledFunctions {
		for _, t := range s.Triggers {
			if trigger == t {
				found = true
				fn = s
			}
		}
	}

	if !found {
		logger.Warningf(e, "command %s not found", trigger)
		f.Replyf(e, "Command %s not found. See %s for a list of available commands.", style.Bold(trigger), style.Italics(fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, f.functionConfig(helpFunctionName).Triggers[0])))
		return
	}

	if len(fn.Triggers) == 0 {
		return
	}

	reply := make([]string, 0)
	reply = append(reply, fmt.Sprintf("%s: %s", style.Bold(style.Underline(trigger)), fn.Description))

	slices.SortFunc(fn.Triggers, func(a, b string) int {
		if len(a) != len(b) {
			return len(a) - len(b)
		}
		return strings.Compare(a, b)
	})

	if len(fn.Usages) > 0 {
		usages := ""
		for _, u := range fn.Usages {
			if len(fn.Triggers) > 0 {
				if len(usages) > 0 {
					usages += ", "
				}
				usages += fmt.Sprintf(u, fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, fn.Triggers[0]))
			}
		}

		reply = append(reply, fmt.Sprintf("Usage: %s", style.Italics(usages)))
	}

	footer := ""
	if len(fn.Role) > 0 && len(fn.ChannelStatus) > 0 {
		footer = fmt.Sprintf("Requires %s role or %s status or greater.", fn.Role, irc.StatusName(fn.ChannelStatus))
	} else if len(fn.Role) > 0 {
		footer = fmt.Sprintf("Requires %s role.", fn.Role)
	} else if len(fn.ChannelStatus) > 0 {
		footer = fmt.Sprintf("Requires %s status or greater.", irc.StatusName(fn.ChannelStatus))
	}

	if fn.DenyPrivateMessages {
		if len(footer) > 0 {
			footer += " "
		}
		footer += "Must be used in a channel."
	}

	if len(footer) > 0 {
		reply = append(reply, footer)
	}

	f.SendMessages(e, e.ReplyTarget(), reply)
}
