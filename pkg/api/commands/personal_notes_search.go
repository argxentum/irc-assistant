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

const PersonalNotesSearchCommandName = "search_personal_notes"
const maxPersonalNotesToShow = 3
const personalNoteMaxLength = 300
const personalNoteListingContentLength = 64

type PersonalNotesSearchCommand struct {
	*commandStub
}

func NewPersonalNotesSearchCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &PersonalNotesSearchCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *PersonalNotesSearchCommand) Name() string {
	return PersonalNotesSearchCommandName
}

func (c *PersonalNotesSearchCommand) Description() string {
	return "Searches your saved personal notes."
}

func (c *PersonalNotesSearchCommand) Triggers() []string {
	return []string{"pnsearch", "pns"}
}

func (c *PersonalNotesSearchCommand) Usages() []string {
	return []string{"%s <search>", "%s <id>"}
}

func (c *PersonalNotesSearchCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *PersonalNotesSearchCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

var personalNoteIDRegex = regexp.MustCompile(`^([a-zA-Z0-9]+)$`)

func (c *PersonalNotesSearchCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	nick := e.From
	tokens := Tokens(e.Message())

	if len(tokens) == 2 && personalNoteIDRegex.MatchString(tokens[1]) {
		n, err := repository.GetPersonalNote(e, nick, tokens[1])
		if err != nil {
			logger.Errorf(e, "Error searching for personal note: %v", err)
			c.Replyf(e, "Sorry, I ran into an error searching for personal note %s.", style.Bold(tokens[1]))
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

	keywords := text.ParseKeywords(input)

	var notes []*models.PersonalNote
	var err error
	if len(keywords) > 0 {
		notes, err = repository.GetPersonalNotesMatchingKeywords(e, e.From, keywords)
	} else if len(url) > 0 {
		notes, err = repository.GetPersonalNotesMatchingSource(e, e.From, url)
	} else {
		notes, err = repository.GetPersonalNotes(e, e.From)
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

	c.SendPersonalNotes(e, notes)
}

func createPersonalNoteOutputMessages(e *irc.Event, nick string, n *models.PersonalNote) []string {
	if len(n.Content) > personalNoteMaxLength {
		n.Content = n.Content[:personalNoteMaxLength] + "..."
	}

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s %s (%s, %s • %s)", "\U0001F5D2\uFE0F", style.Bold(n.Content), nick, elapse.PastTimeDescription(n.NotedAt), n.ID))

	if len(n.Source) > 0 {
		messages = append(messages, n.Source)

		source, err := repository.FindSource(n.Source)
		if err != nil {
			log.Logger().Errorf(nil, "error finding source, %s", err)
		}

		if source != nil {
			messages = append(messages, repository.ShortSourceSummary(source))
		}
	}

	return messages
}

func (c *PersonalNotesSearchCommand) SendPersonalNote(e *irc.Event, n *models.PersonalNote) {
	c.SendMessages(e, e.ReplyTarget(), createPersonalNoteOutputMessages(e, e.From, n))
}

func (c *PersonalNotesSearchCommand) SendPersonalNotes(e *irc.Event, notes []*models.PersonalNote) {
	if len(notes) == 1 {
		c.SendPersonalNote(e, notes[0])
		return
	}

	qty := "notes"
	if len(notes) == 1 {
		qty = "note"
	}

	if len(notes) > maxPersonalNotesToShow {
		c.Replyf(e, fmt.Sprintf("Found %s matching %s. Displaying %s best matches:", style.Bold(fmt.Sprintf("%d", len(notes))), qty, style.Bold(fmt.Sprintf("%d", maxPersonalNotesToShow))))
	} else {
		c.Replyf(e, fmt.Sprintf("Found %s matching %s:", style.Bold(fmt.Sprintf("%d", len(notes))), qty))
	}

	messages := make([]string, 0)
	shown := 0
	for i := len(notes) - 1; i >= 0; i-- {
		if shown >= maxPersonalNotesToShow {
			break
		}

		n := notes[i]
		if len(n.Content) > personalNoteListingContentLength {
			n.Content = n.Content[:personalNoteListingContentLength] + "..."
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
