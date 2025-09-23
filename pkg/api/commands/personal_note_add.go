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

const PersonalNoteAddCommandName = "add_personal_note"

type PersonalNoteAddCommand struct {
	*commandStub
}

func NewPersonalNoteAddCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &PersonalNoteAddCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *PersonalNoteAddCommand) Name() string {
	return PersonalNoteAddCommandName
}

func (c *PersonalNoteAddCommand) Description() string {
	return "Saves a new personal note."
}

func (c *PersonalNoteAddCommand) Triggers() []string {
	return []string{"pn"}
}

func (c *PersonalNoteAddCommand) Usages() []string {
	return []string{"%s <note>"}
}

func (c *PersonalNoteAddCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *PersonalNoteAddCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *PersonalNoteAddCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	nick := e.From
	tokens := Tokens(e.Message())

	// attempt to perform a lookup if the user accidentally provided an ID
	if len(tokens) == 2 && personalNoteIDRegex.MatchString(tokens[1]) {
		n, err := repository.GetPersonalNote(e, nick, tokens[1])
		if err != nil {
			logger.Errorf(e, "Error searching for personal note: %v", err)
			c.Replyf(e, "Sorry, I ran into an error.")
			return
		}

		if n != nil {
			c.SendMessages(e, e.ReplyTarget(), createPersonalNoteOutputMessages(e, nick, n))
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

	n := models.NewPersonalNote(input, url)

	if err := repository.AddPersonalNote(e, e.From, n); err != nil {
		logger.Errorf(e, "Error adding personal note: %v", err)
		c.Replyf(e, "Sorry, I couldn't save the personal note.")
		return
	}

	c.Replyf(e, "Personal note %s saved.", style.Bold(n.ID))
}
