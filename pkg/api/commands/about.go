package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const aboutCommandName = "about"

type aboutCommand struct {
	*commandStub
}

func NewAboutCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &aboutCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *aboutCommand) Name() string {
	return aboutCommandName
}

func (c *aboutCommand) Description() string {
	return "Shows information about the bot."
}

func (c *aboutCommand) Triggers() []string {
	return []string{"about"}
}

func (c *aboutCommand) Usages() []string {
	return []string{"%s"}
}

func (c *aboutCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *aboutCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *aboutCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	message := "Version 0.1. Source: https://github.com/argxentum/irc-assistant."
	c.SendMessage(e, e.ReplyTarget(), message)
}
