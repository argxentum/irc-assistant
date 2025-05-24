package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
)

const ShortcutAddCommandName = "shortcut_add"

type ShortcutAddCommand struct {
	*commandStub
}

func NewShortcutAddCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &ShortcutAddCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *ShortcutAddCommand) Name() string {
	return ShortcutAddCommandName
}

func (c *ShortcutAddCommand) Description() string {
	return "Creates a shortcut to the specified URL."
}

func (c *ShortcutAddCommand) Triggers() []string {
	return []string{"shortcut", "scadd", "sc"}
}

func (c *ShortcutAddCommand) Usages() []string {
	return []string{"%s <url>"}
}

func (c *ShortcutAddCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *ShortcutAddCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *ShortcutAddCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	url := tokens[1]

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), url)

	if !urlRegex.MatchString(url) {
		logger.Debugf(e, "invalid URL: %s", url)
		c.Replyf(e, "Sorry, but I can't create a shortcut for %s", url)
		return
	}

	s, err := repository.GetShortcut(url, url)
	if err != nil {
		logger.Errorf(e, "failed to create shortcut %s: %v", url, err)
		c.Replyf(e, "Failed to create shortcut for %s", url)
		return
	}

	scu := fmt.Sprintf(shortcutURLPattern, c.cfg.Web.ExternalRootURL) + s.ID
	c.SendMessage(e, e.ReplyTarget(), scu)
}
