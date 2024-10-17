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
	*functionStub
}

func NewHelpFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) Function {
	return &helpFunction{
		functionStub: defaultFunctionStub(ctx, cfg, irc),
	}
}

func (f *helpFunction) Name() string {
	return helpFunctionName
}

func (f *helpFunction) Description() string {
	return "Displays help for the given command."
}

func (f *helpFunction) Triggers() []string {
	return []string{"help"}
}

func (f *helpFunction) Usages() []string {
	return []string{"%s", "%s <command>"}
}

func (f *helpFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *helpFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 0)
}

func (f *helpFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), e.Message())

	// if no command is specified, list all available commands
	if len(tokens) == 1 {
		reply := make([]string, 0)
		reply = append(reply, fmt.Sprintf("%s: %s", style.Bold(style.Underline(f.Name())), f.Description()))

		// create map of function name to slice of current user authorization and allowed user status
		commands := make([]string, 0)
		for _, fn := range registry.Functions() {
			fnt := ""
			for i, t := range fn.Triggers() {
				if len(fnt) > 0 {
					fnt += "/"
				}

				if i == len(fn.Triggers())-1 && (len(fn.Authorizer().RequiredRole()) > 0 || len(fn.Authorizer().RequiredChannelStatus()) > 0) {
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
		for _, u := range f.Usages() {
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

	var fn Function
	for _, s := range registry.Functions() {
		for _, t := range s.Triggers() {
			if trigger == t {
				fn = s
			}
		}
	}

	if fn == nil {
		logger.Warningf(e, "command %s not found", trigger)
		f.Replyf(e, "Command %s not found. See %s for a list of available commands.", style.Bold(trigger), style.Italics(fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, registry.Function(helpFunctionName).Triggers()[0])))
		return
	}

	if len(fn.Triggers()) == 0 {
		return
	}

	extraTriggers := make([]string, 0)
	for _, t := range fn.Triggers() {
		if t != trigger {
			extraTriggers = append(extraTriggers, t)
		}
	}
	slices.SortFunc(extraTriggers, func(a, b string) int {
		if len(a) != len(b) {
			return len(a) - len(b)
		}
		return strings.Compare(a, b)
	})

	extra := ""
	for _, t := range extraTriggers {
		if len(extra) > 0 {
			extra += ", "
		}
		extra += style.Bold(style.Underline(t))
	}

	if len(extraTriggers) > 0 {
		extra = fmt.Sprintf(" (or %s)", extra)
	}

	reply := make([]string, 0)
	reply = append(reply, fmt.Sprintf("%s%s: %s", style.Bold(style.Underline(trigger)), extra, fn.Description()))

	if len(fn.Usages()) > 0 {
		usages := ""
		for _, u := range fn.Usages() {
			if len(fn.Triggers()) > 0 {
				if len(usages) > 0 {
					usages += ", "
				}
				usages += fmt.Sprintf(u, fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, trigger))
			}
		}

		reply = append(reply, fmt.Sprintf("Usage: %s", style.Italics(usages)))
	}

	footer := ""
	if len(fn.Authorizer().RequiredRole()) > 0 && len(fn.Authorizer().RequiredChannelStatus()) > 0 {
		footer = fmt.Sprintf("Requires %s role or %s status or greater.", fn.Authorizer().RequiredRole(), irc.StatusName(fn.Authorizer().RequiredChannelStatus()))
	} else if len(fn.Authorizer().RequiredRole()) > 0 {
		footer = fmt.Sprintf("Requires %s role.", fn.Authorizer().RequiredRole())
	} else if len(fn.Authorizer().RequiredChannelStatus()) > 0 {
		footer = fmt.Sprintf("Requires %s status or greater.", irc.StatusName(fn.Authorizer().RequiredChannelStatus()))
	}

	if !fn.AllowedInPrivateMessages() {
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
