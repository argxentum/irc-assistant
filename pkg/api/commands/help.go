package commands

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

const HelpCommandName = "help"

type HelpCommand struct {
	*commandStub
}

func NewHelpCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &HelpCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *HelpCommand) Name() string {
	return HelpCommandName
}

func (c *HelpCommand) Description() string {
	return "Displays help for the given command."
}

func (c *HelpCommand) Triggers() []string {
	return []string{"help"}
}

func (c *HelpCommand) Usages() []string {
	return []string{"%s", "%s <command>"}
}

func (c *HelpCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *HelpCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *HelpCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), e.Message())

	// if no command is specified, list all available commands
	if len(tokens) == 1 {
		reply := make([]string, 0)
		reply = append(reply, fmt.Sprintf("%s: %s", style.Bold(style.Underline(c.Name())), c.Description()))

		// create map of command name to slice of current user authorization and allowed user status
		commands := make([]string, 0)
		for _, cmd := range registry.Commands() {
			cmdt := ""
			for i, t := range cmd.Triggers() {
				if len(cmdt) > 0 {
					cmdt += "/"
				}

				if i == len(cmd.Triggers())-1 && (len(cmd.Authorizer().RequiredRole()) > 0 || len(cmd.Authorizer().RequiredChannelStatus()) > 0) {
					cmdt += fmt.Sprintf("%s*", strings.TrimPrefix(t, c.cfg.Commands.Prefix))
				} else {
					cmdt += strings.TrimPrefix(t, c.cfg.Commands.Prefix)
				}
			}
			commands = append(commands, cmdt)
		}
		slices.Sort(commands)

		cmds := ""
		for _, cmd := range commands {
			if len(cmds) > 0 {
				cmds += ", "
			}
			cmds += cmd
		}
		reply = append(reply, fmt.Sprintf("%s: %s (* requires authorization)", style.Underline("Commands"), cmds))
		usages := ""
		for _, u := range c.Usages() {
			if len(usages) > 0 {
				usages += ", "
			}
			usages += fmt.Sprintf(u, fmt.Sprintf("%shelp", c.cfg.Commands.Prefix))
		}
		reply = append(reply, fmt.Sprintf("%s: %s", style.Underline("Usage"), style.Italics(usages)))

		c.SendMessages(e, e.ReplyTarget(), reply)
		return
	}

	trigger := strings.TrimPrefix(tokens[1], c.cfg.Commands.Prefix)

	var cmd Command
	for _, s := range registry.Commands() {
		for _, t := range s.Triggers() {
			if trigger == t {
				cmd = s
			}
		}
	}

	if cmd == nil {
		logger.Warningf(e, "command %s not found", trigger)
		c.Replyf(e, "Command %s not found. See %s for a list of available commands.", style.Bold(trigger), style.Italics(fmt.Sprintf("%s%s", c.cfg.Commands.Prefix, registry.Command(HelpCommandName).Triggers()[0])))
		return
	}

	if len(cmd.Triggers()) == 0 {
		return
	}

	extraTriggers := make([]string, 0)
	for _, t := range cmd.Triggers() {
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
	reply = append(reply, fmt.Sprintf("%s%s: %s", style.Bold(style.Underline(trigger)), extra, cmd.Description()))

	if len(cmd.Usages()) > 0 {
		usages := ""
		for _, u := range cmd.Usages() {
			if len(cmd.Triggers()) > 0 {
				if len(usages) > 0 {
					usages += ", "
				}
				usages += fmt.Sprintf(u, fmt.Sprintf("%s%s", c.cfg.Commands.Prefix, trigger))
			}
		}

		reply = append(reply, fmt.Sprintf("Usage: %s", style.Italics(usages)))
	}

	footer := ""
	if len(cmd.Authorizer().RequiredRole()) > 0 && len(cmd.Authorizer().RequiredChannelStatus()) > 0 {
		footer = fmt.Sprintf("Requires %s role or %s status or greater.", cmd.Authorizer().RequiredRole(), irc.StatusName(cmd.Authorizer().RequiredChannelStatus()))
	} else if len(cmd.Authorizer().RequiredRole()) > 0 {
		footer = fmt.Sprintf("Requires %s role.", cmd.Authorizer().RequiredRole())
	} else if len(cmd.Authorizer().RequiredChannelStatus()) > 0 {
		footer = fmt.Sprintf("Requires %s status or greater.", irc.StatusName(cmd.Authorizer().RequiredChannelStatus()))
	}

	if !cmd.AllowedInPrivateMessages() {
		if len(footer) > 0 {
			footer += " "
		}
		footer += "Must be used in a channel."
	}

	if len(footer) > 0 {
		reply = append(reply, footer)
	}

	c.SendMessages(e, e.ReplyTarget(), reply)
}
