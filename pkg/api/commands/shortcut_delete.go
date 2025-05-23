package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const ShortcutDeleteCommandName = "shortcut_delete"

type ShortcutDeleteCommand struct {
	*commandStub
}

func NewShortcutDeleteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &ShortcutDeleteCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *ShortcutDeleteCommand) Name() string {
	return ShortcutDeleteCommandName
}

func (c *ShortcutDeleteCommand) Description() string {
	return "Deletes the shortcut with the specified ID."
}

func (c *ShortcutDeleteCommand) Triggers() []string {
	return []string{"scdel"}
}

func (c *ShortcutDeleteCommand) Usages() []string {
	return []string{"%s <id>"}
}

func (c *ShortcutDeleteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *ShortcutDeleteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *ShortcutDeleteCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	id := tokens[1]

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), id)

	err := repository.RemoveShortcut(id)
	if err != nil {
		logger.Errorf(e, "failed to delete shortcut %s: %v", id, err)
		c.Replyf(e, "Failed to delete shortcut %s", id)
		return
	}

	c.Replyf(e, "Deleted shortcut %s", id)
}
