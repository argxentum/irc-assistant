package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
)

const CommunityNoteGetCommandName = "get_community_note"

type CommunityNoteGetCommand struct {
	*commandStub
}

func NewCommunityNoteGetCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &CommunityNoteGetCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *CommunityNoteGetCommand) Name() string {
	return CommunityNoteGetCommandName
}

func (c *CommunityNoteGetCommand) Description() string {
	return "Gets a community note."
}

func (c *CommunityNoteGetCommand) Triggers() []string {
	return []string{"cn"}
}

func (c *CommunityNoteGetCommand) Usages() []string {
	return []string{"%s [<channel>] <id>"}
}

func (c *CommunityNoteGetCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *CommunityNoteGetCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *CommunityNoteGetCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if e.IsPrivateMessage() {
		channel = tokens[1]
		if len(tokens) < 3 || !irc.IsChannel(channel) {
			c.Replyf(e, "Please specify a channel: %s", style.Italics(fmt.Sprintf("%s <channel> <id>", tokens[0])))
			return
		}

		updatedTokens := make([]string, 0)
		for i, token := range tokens {
			if i != 1 {
				updatedTokens = append(updatedTokens, token)
			}
		}
		tokens = updatedTokens
	}

	id := tokens[1]

	n, err := repository.CommunityNote(e, channel, id)
	if err != nil {
		logger.Errorf(e, "error searching for community note: %v", err)
		c.Replyf(e, "Sorry, I couldn't find community note %s.", style.Bold(id))
		return
	}

	if n != nil {
		c.SendMessages(e, e.ReplyTarget(), createCommunityNoteOutputMessages(e, n))
		return
	}
}
