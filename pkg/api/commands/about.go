package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const AboutCommandName = "about"

type AboutCommand struct {
	*commandStub
}

func NewAboutCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &AboutCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *AboutCommand) Name() string {
	return AboutCommandName
}

func (c *AboutCommand) Description() string {
	return "Shows information about the bot."
}

func (c *AboutCommand) Triggers() []string {
	return []string{"about"}
}

func (c *AboutCommand) Usages() []string {
	return []string{"%s"}
}

func (c *AboutCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *AboutCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *AboutCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	message := "Version 0.1. Source: https://github.com/argxentum/irc-assistant."
	c.SendMessage(e, e.ReplyTarget(), message)
}
