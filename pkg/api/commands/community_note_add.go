package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"strings"
)

const CommunityNoteAddCommandName = "add_community_note"

type CommunityNoteAddCommand struct {
	*commandStub
}

func NewCommunityNoteAddCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &CommunityNoteAddCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *CommunityNoteAddCommand) Name() string {
	return CommunityNoteAddCommandName
}

func (c *CommunityNoteAddCommand) Description() string {
	return "Creates a community note."
}

func (c *CommunityNoteAddCommand) Triggers() []string {
	return []string{"cnadd"}
}

func (c *CommunityNoteAddCommand) Usages() []string {
	return []string{"%s [<channel>] <source> <counter-source> <note>"}
}

func (c *CommunityNoteAddCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *CommunityNoteAddCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 3)
}

func (c *CommunityNoteAddCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", c.Name(), e.From, e.ReplyTarget())
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if e.IsPrivateMessage() {
		channel = tokens[1]
		if len(tokens) < 3 || !irc.IsChannel(channel) {
			c.Replyf(e, "Please specify a channel: %s", style.Italics(fmt.Sprintf("%s <channel> <id>", tokens[0])))
			return
		}
		tokens = tokens[2:]
	}

	source := strings.TrimSpace(strings.ToLower(tokens[1]))
	counterSource := strings.TrimSpace(strings.ToLower(tokens[2]))
	note := strings.TrimSpace(strings.Join(tokens[3:], " "))

	n, err := repository.GetCommunityNoteForSource(e, channel, source)
	if err != nil {
		logger.Errorf(e, "error searching for community note: %v", err)
	}

	if n == nil {
		logger.Debugf(e, "creating new community note for source %s", source)
		n = models.NewCommunityNote(note, source, e.From, counterSource)

		if err = repository.CreateCommunityNote(e, channel, n); err != nil {
			logger.Errorf(e, "error adding personal note: %v", err)
			c.Replyf(e, "Sorry, I couldn't create the personal note.")
			return
		}

		c.Replyf(e, "Created community note %s for source %s", style.Bold(n.ID), source)
		return
	}

	logger.Debugf(e, "updating community note %s for source %s", n.ID, source)

	counterSourceExists := false
	for _, cs := range n.CounterSources {
		if cs == counterSource {
			counterSourceExists = true
			break
		}
	}

	if !counterSourceExists {
		logger.Debugf(e, "adding new counter-source %s for source %s", counterSource, source)
		n.CounterSources = append(n.CounterSources, counterSource)
	}

	n.Content = note

	err = repository.UpdateCommunityNote(e, channel, n)
	if err != nil {
		logger.Errorf(e, "error updating community note: %v", err)
		c.Replyf(e, "Sorry, I couldn't update the community note.")
		return
	}

	c.Replyf(e, "Community note %s updated for %s.", style.Bold(n.ID), source)
}
