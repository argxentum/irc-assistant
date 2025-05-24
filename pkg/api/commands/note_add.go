package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"strings"
)

const NoteAddCommandName = "add_note"

type NoteAddCommand struct {
	*commandStub
}

func NewNoteAddCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &NoteAddCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *NoteAddCommand) Name() string {
	return NoteAddCommandName
}

func (c *NoteAddCommand) Description() string {
	return "Adds a new note."
}

func (c *NoteAddCommand) Triggers() []string {
	return []string{"note", "n"}
}

func (c *NoteAddCommand) Usages() []string {
	return []string{"%s <note>"}
}

func (c *NoteAddCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *NoteAddCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *NoteAddCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	nick := e.From
	tokens := Tokens(e.Message())

	// attempt to perform a lookup if the user accidentally provided an ID
	if len(tokens) == 2 && noteIDRegex.MatchString(tokens[1]) {
		n, err := repository.GetUserNote(e, nick, tokens[1])
		if err != nil {
			logger.Errorf(e, "Error searching for note: %v", err)
			c.Replyf(e, "Sorry, I ran into an error.")
			return
		}

		if n != nil {
			c.SendMessages(e, e.ReplyTarget(), createNoteOutputMessages(e, nick, n))
			return
		}
	}

	input := strings.Join(tokens[1:], " ")
	url := ""
	if urlRegex.MatchString(input) {
		url = urlRegex.FindString(input)
		input = strings.ReplaceAll(input, url, "")
		input = strings.TrimSpace(input)
	}

	n := models.NewNote(input, url)

	if err := repository.AddUserNote(e, e.From, n); err != nil {
		logger.Errorf(e, "Error adding note: %v", err)
		c.Replyf(e, "Sorry, I couldn't add the note.")
		return
	}

	c.Replyf(e, "Note %s saved.", style.Bold(n.ID))
}
