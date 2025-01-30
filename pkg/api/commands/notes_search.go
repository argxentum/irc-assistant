package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"regexp"
	"strings"
)

const NotesSearchCommandName = "notes_search"
const maxNotesToShow = 3
const noteMaxLength = 300
const noteListingContentLength = 64

type NotesSearchCommand struct {
	*commandStub
}

func NewNotesSearchCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &NotesSearchCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *NotesSearchCommand) Name() string {
	return NotesSearchCommandName
}

func (c *NotesSearchCommand) Description() string {
	return "Searches your stored notes."
}

func (c *NotesSearchCommand) Triggers() []string {
	return []string{"notes", "ns"}
}

func (c *NotesSearchCommand) Usages() []string {
	return []string{"%s <search>", "%s <id>"}
}

func (c *NotesSearchCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *NotesSearchCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

var noteIDRegex = regexp.MustCompile(`^([a-zA-Z0-9]+)$`)

func (c *NotesSearchCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	nick := e.From
	tokens := Tokens(e.Message())

	if len(tokens) == 2 && noteIDRegex.MatchString(tokens[1]) {
		n, err := repository.GetUserNote(e, nick, tokens[1])
		if err != nil {
			logger.Errorf(e, "Error searching for note: %v", err)
			c.Replyf(e, "Sorry, I ran into an error searching for note %s.", style.Bold(tokens[1]))
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

	keywords := text.ParseKeywords(input)

	var notes []*models.Note
	var err error
	if len(keywords) > 0 {
		notes, err = repository.GetUserNotesMatchingKeywords(e, e.From, keywords)
	} else if len(url) > 0 {
		notes, err = repository.GetUserNotesMatchingSource(e, e.From, url)
	} else {
		notes, err = repository.GetUserNotes(e, e.From)
	}

	if err != nil {
		logger.Debugf(e, "unable to retrieve notes: %s", err)
		c.Replyf(e, "Sorry, but I ran into an issue searching for notes.")
		return
	}

	if len(notes) == 0 {
		logger.Debugf(e, "no notes found")
		c.Replyf(e, "No notes found.")
		return
	}

	c.SendNotes(e, notes)
}

func createNoteOutputMessages(e *irc.Event, nick string, n *models.Note) []string {
	if len(n.Content) > noteMaxLength {
		n.Content = n.Content[:noteMaxLength] + "..."
	}

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("🗒 %s (%s, %s • %s)", style.Bold(n.Content), nick, elapse.PastTimeDescription(n.NotedAt), n.ID))

	if len(n.Source) > 0 {
		messages = append(messages, n.Source)

		if bias, ok := repository.GetBiasResult(e, n.Source, false); ok {
			messages = append(messages, bias.ShortDescription())
		}
	}

	return messages
}

func (c *NotesSearchCommand) SendNote(e *irc.Event, n *models.Note) {
	c.SendMessages(e, e.ReplyTarget(), createNoteOutputMessages(e, e.From, n))
}

func (c *NotesSearchCommand) SendNotes(e *irc.Event, notes []*models.Note) {
	if len(notes) == 1 {
		c.SendNote(e, notes[0])
		return
	}

	if len(notes) > maxNotesToShow {
		c.Replyf(e, fmt.Sprintf("Found %s matching notes (only showing %s)", style.Bold(fmt.Sprintf("%d", len(notes))), style.Bold(fmt.Sprintf("%d", maxNotesToShow))))
	} else {
		c.Replyf(e, fmt.Sprintf("Found %s matching notes", style.Bold(fmt.Sprintf("%d", len(notes)))))
	}

	messages := make([]string, 0)
	shown := 0
	for i := len(notes) - 1; i >= 0; i-- {
		if shown >= maxNotesToShow {
			break
		}

		n := notes[i]
		if len(n.Content) > noteListingContentLength {
			n.Content = n.Content[:noteListingContentLength] + "..."
		}

		note := n.Content
		if len(note) == 0 && len(n.Source) > 0 {
			note = n.Source
		}

		messages = append(messages, fmt.Sprintf("%s: %s", style.Bold(n.ID), note))
		shown++
	}

	c.SendMessages(e, e.ReplyTarget(), messages)
}
