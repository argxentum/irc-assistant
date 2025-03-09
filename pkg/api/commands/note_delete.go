package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const NoteDeleteCommandName = "delete_note"

type NoteDeleteCommand struct {
	*commandStub
}

func NewNoteDeleteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &NoteDeleteCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *NoteDeleteCommand) Name() string {
	return NoteDeleteCommandName
}

func (c *NoteDeleteCommand) Description() string {
	return "Deletes a note."
}

func (c *NoteDeleteCommand) Triggers() []string {
	return []string{"notedel", "nd"}
}

func (c *NoteDeleteCommand) Usages() []string {
	return []string{"%s <id>"}
}

func (c *NoteDeleteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *NoteDeleteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *NoteDeleteCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	nick := e.From
	tokens := Tokens(e.Message())
	id := tokens[1]

	if err := repository.DeleteUserNote(e, nick, id); err != nil {
		c.Replyf(e, "Error deleting note: %v", err)
		return
	}

	c.Replyf(e, "Note %s deleted.", style.Bold(id))
}
