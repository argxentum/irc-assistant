package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
)

const PersonalNoteDeleteCommandName = "delete_personal_note"

type PersonalNoteDeleteCommand struct {
	*commandStub
}

func NewPersonalNoteDeleteCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &PersonalNoteDeleteCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *PersonalNoteDeleteCommand) Name() string {
	return PersonalNoteDeleteCommandName
}

func (c *PersonalNoteDeleteCommand) Description() string {
	return "Deletes one of your personal notes."
}

func (c *PersonalNoteDeleteCommand) Triggers() []string {
	return []string{"pndel", "pnd"}
}

func (c *PersonalNoteDeleteCommand) Usages() []string {
	return []string{"%s <id>"}
}

func (c *PersonalNoteDeleteCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *PersonalNoteDeleteCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *PersonalNoteDeleteCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	nick := e.From
	tokens := Tokens(e.Message())
	id := tokens[1]

	if err := repository.DeletePersonalNote(e, nick, id); err != nil {
		c.Replyf(e, "Error deleting personal note %s", style.Bold(id))
		return
	}

	c.Replyf(e, "Personal note %s deleted.", style.Bold(id))
}
