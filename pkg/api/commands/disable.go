package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"slices"
	"strings"
)

const DisableCommandName = "disable"

type DisableCommand struct {
	*commandStub
}

func NewDisableCommand(ctx context.Context, cfg *config.Config, ircSvc irc.IRC) Command {
	return &DisableCommand{
		commandStub: newCommandStub(ctx, cfg, ircSvc, RoleAdmin, irc.ChannelStatusNormal),
	}
}

func (c *DisableCommand) Name() string {
	return DisableCommandName
}

func (c *DisableCommand) Description() string {
	return "Disables the specified command."
}

func (c *DisableCommand) Triggers() []string {
	return []string{"disable"}
}

func (c *DisableCommand) Usages() []string {
	return []string{"%s [<channel>] <command> [<command> ...]"}
}

func (c *DisableCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *DisableCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *DisableCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := ""
	input := tokens[1:]
	if irc.IsChannel(tokens[1]) {
		channel = tokens[1]
		input = tokens[2:]
	}

	if len(channel) == 0 {
		if e.IsPrivateMessage() {
			c.Replyf(e, "Please specify a channel: %s", style.Italics(fmt.Sprintf("%s <channel> <command>", tokens[0])))
			return
		} else {
			channel = e.ReplyTarget()
		}
	}

	message := strings.Join(input, " ")
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), message)

	commands := make([]string, 0)

	for _, t := range input {
		t = strings.TrimPrefix(strings.ToLower(t), c.cfg.Commands.Prefix)
		for k, v := range registry.Commands() {
			if k == t || slices.Contains(v.Triggers(), t) {
				commands = append(commands, k)
			}
		}
	}

	if len(commands) == 0 {
		c.Replyf(e, "Disable failed for: %s", style.Bold(message))
		return
	}

	ch, err := repository.GetChannel(e, channel)
	if err != nil {
		log.Logger().Errorf(e, "error retrieving channel, %s", err)
		return
	}

	if ch == nil {
		log.Logger().Errorf(e, "channel %s not found", channel)
		return
	}

	processed := make([]string, 0)
	for _, cmd := range commands {
		if slices.Contains(ch.DisabledCommands, cmd) {
			continue
		}

		ch.DisabledCommands = append(ch.DisabledCommands, cmd)

		if err = repository.UpdateChannelDisabledCommands(e, ch); err != nil {
			log.Logger().Errorf(e, "error updating channel, %s", err)
			return
		}

		processed = append(processed, cmd)
	}

	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("Disabled: %s", style.Bold(strings.Join(processed, ", "))))
}
